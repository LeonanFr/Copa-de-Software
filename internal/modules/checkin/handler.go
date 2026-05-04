package checkin

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

	router.HandleFunc("/checkin", h.listar).Methods("GET")
	router.HandleFunc("/checkin/participant/{id}", h.checkinParticipant).Methods("POST")
	router.HandleFunc("/checkin/manual", h.adicionarOuvinte).Methods("POST")
	router.HandleFunc("/checkin/{id}", h.remover).Methods("DELETE")
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	checkins, err := h.service.ListarCheckins(r.Context())
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao listar check-ins", err))
		return
	}
	shared.RespondJSON(w, http.StatusOK, checkins)
}

func (h *Handler) checkinParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	participantID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id de participante inválido", err))
		return
	}

	checkin, err := h.service.CheckinParticipant(r.Context(), participantID)
	if err != nil {
		switch {
		case errors.Is(err, participants.ErrParticipantNotFound):
			shared.RespondError(w, shared.NewNotFoundError("participante não encontrado", nil))
		case errors.Is(err, ErrCheckinAlreadyExists):
			shared.RespondError(w, shared.NewConflictError("participante já realizou check-in", nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao realizar check-in", err))
		}
		return
	}

	shared.RespondJSON(w, http.StatusCreated, checkin)
}

func (h *Handler) adicionarOuvinte(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nome string `json:"nome"`
	}
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}
	if req.Nome == "" {
		shared.RespondError(w, shared.NewBadRequestError("nome é obrigatório", nil))
		return
	}

	checkin, err := h.service.AdicionarOuvinte(r.Context(), req.Nome)
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao adicionar ouvinte", err))
		return
	}

	shared.RespondJSON(w, http.StatusCreated, checkin)
}

func (h *Handler) remover(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	checkinID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id de check-in inválido", err))
		return
	}

	if err := h.service.RemoverCheckin(r.Context(), checkinID); err != nil {
		if errors.Is(err, ErrCheckinNotFound) {
			shared.RespondError(w, shared.NewNotFoundError("check-in não encontrado", nil))
		} else {
			shared.RespondError(w, shared.NewInternalServerError("erro ao remover check-in", err))
		}
		return
	}

	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "removido"})
}
