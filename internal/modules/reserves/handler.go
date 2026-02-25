package reserves

import (
	"errors"
	"net/http"

	"copasoftware/internal/modules/participants"
	"copasoftware/internal/shared"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler struct {
	service *Service
}

func NewHandler(router *mux.Router, service *Service) {
	h := &Handler{service: service}

	router.HandleFunc("/reserves", h.list).Methods("GET")
	router.HandleFunc("/reserves", h.add).Methods("POST")
	router.HandleFunc("/reserves/{id}", h.remove).Methods("DELETE")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	entries, err := h.service.ListReserves(r.Context())
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao listar reserva", err))
		return
	}
	shared.RespondJSON(w, http.StatusOK, entries)
}

type addRequest struct {
	ParticipantID string `json:"participantId"`
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	var req addRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}

	pid, err := primitive.ObjectIDFromHex(req.ParticipantID)
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id de participante inválido", err))
		return
	}

	err = h.service.AddToReserve(r.Context(), pid)
	if err != nil {
		switch {
		case errors.Is(err, participants.ErrParticipantNotFound):
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		case errors.Is(err, ErrAlreadyInReserve):
			shared.RespondError(w, shared.NewConflictError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao adicionar à reserva", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusCreated, map[string]string{"status": "adicionado à reserva"})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	err = h.service.RemoveFromReserve(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrReserveEntryNotFound) {
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		} else {
			shared.RespondError(w, shared.NewInternalServerError("erro ao remover da reserva", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "removido da reserva"})
}
