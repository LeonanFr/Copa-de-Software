package draw

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"copasoftware/internal/modules/reserves"
	"copasoftware/internal/modules/teamnames"
	"copasoftware/internal/shared"

	"github.com/gorilla/mux"
)

type Handler struct {
	service    *Service
	reserveSvc *reserves.Service
}

func NewHandler(router *mux.Router, service *Service, reserveSvc *reserves.Service) {
	h := &Handler{
		service:    service,
		reserveSvc: reserveSvc,
	}

	router.HandleFunc("/draw", h.runDraw).Methods("POST")
}

func (h *Handler) runDraw(w http.ResponseWriter, r *http.Request) {
	finalParam := r.URL.Query().Get("final")
	isFinal := false
	if finalParam != "" {
		var err error
		isFinal, err = strconv.ParseBool(finalParam)
		if err != nil {
			shared.RespondError(w, shared.NewBadRequestError("parâmetro 'final' deve ser true ou false", err))
			return
		}
	}

	result, err := h.service.RunDraw(r.Context(), isFinal)
	if err != nil {
		if errors.Is(err, teamnames.ErrNoNamesAvailable) {
			shared.RespondError(w, shared.NewConflictError("nenhum nome de time disponível para o sorteio", nil))
			return
		}
		shared.RespondError(w, shared.NewInternalServerError("erro ao executar sorteio", err))
		return
	}

	remaining := result.Remaining
	totalEligible := result.TotalEligible

	if len(remaining) == totalEligible && totalEligible > 0 {
		message := "Não foi possível formar novos times com os participantes restantes"
		if isFinal {
			message = "Sorteio final concluído: não há participantes suficientes para formar novos times"
		}
		response := map[string]interface{}{
			"message":   message,
			"remaining": remaining,
		}
		shared.RespondJSON(w, http.StatusOK, response)
		return
	}

	if isFinal && h.reserveSvc != nil && len(remaining) > 0 {
		for _, p := range remaining {
			if err := h.reserveSvc.AddToReserve(r.Context(), p.ID); err != nil {
				log.Printf("erro ao adicionar participante %s à reserva: %v", p.ID.Hex(), err)
			}
		}
	}

	response := map[string]interface{}{
		"message":   "sorteio concluído",
		"remaining": remaining,
	}
	shared.RespondJSON(w, http.StatusOK, response)
}
