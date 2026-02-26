package participants

import (
	"errors"
	"net/http"

	"copasoftware/internal/shared"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *Service
}

func RegisterPublicRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/participants", h.list).Methods("GET")
	router.HandleFunc("/participants/{matricula}", h.getByMatricula).Methods("GET")
}

func RegisterAdminRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/participants/{matricula}", h.update).Methods("PUT")
	router.HandleFunc("/participants/{matricula}/cancel", h.cancel).Methods("POST")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	participants, err := h.service.List(r.Context())
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao listar participantes", err))
		return
	}
	shared.RespondJSON(w, http.StatusOK, participants)
}

func (h *Handler) getByMatricula(w http.ResponseWriter, r *http.Request) {
	matricula := mux.Vars(r)["matricula"]
	if matricula == "" {
		shared.RespondError(w, shared.NewBadRequestError("matrícula não fornecida", nil))
		return
	}

	p, err := h.service.GetByMatricula(r.Context(), matricula)
	if err != nil {
		if errors.Is(err, ErrParticipantNotFound) {
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		} else {
			shared.RespondError(w, shared.NewInternalServerError("erro ao buscar participante", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, p)
}

type updateRequest struct {
	Nome     string `json:"nome"`
	Semestre int    `json:"semestre"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	matricula := mux.Vars(r)["matricula"]
	if matricula == "" {
		shared.RespondError(w, shared.NewBadRequestError("matrícula não fornecida", nil))
		return
	}

	var req updateRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}

	p, err := h.service.UpdateByMatricula(r.Context(), matricula, req.Nome, req.Semestre)
	if err != nil {
		switch {
		case errors.Is(err, ErrParticipantNotFound):
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		case errors.Is(err, ErrInvalidSemester):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao atualizar participante", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, p)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	matricula := mux.Vars(r)["matricula"]
	if matricula == "" {
		shared.RespondError(w, shared.NewBadRequestError("matrícula não fornecida", nil))
		return
	}

	if err := h.service.CancelByMatricula(r.Context(), matricula); err != nil {
		if errors.Is(err, ErrParticipantNotFound) {
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		} else {
			shared.RespondError(w, shared.NewInternalServerError("erro ao cancelar participante", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "cancelado"})
}
