package teams

import (
	"errors"
	"net/http"

	"copasoftware/internal/shared"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler struct {
	service *Service
}

func RegisterPublicRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/teams", h.list).Methods("GET")
	router.HandleFunc("/teams/{id}", h.getByID).Methods("GET")
}

func RegisterAdminRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/teams", h.createManual).Methods("POST")
	router.HandleFunc("/teams/{id}/approve", h.approve).Methods("POST")
	router.HandleFunc("/teams/{id}/reject", h.reject).Methods("POST")
	router.HandleFunc("/teams/{id}/cancel", h.cancel).Methods("POST")
}

type createManualRequest struct {
	ParticipantIDs []string `json:"participantIds"`
}

func (h *Handler) createManual(w http.ResponseWriter, r *http.Request) {
	var req createManualRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}

	if len(req.ParticipantIDs) != 3 {
		shared.RespondError(w, shared.NewBadRequestError("time deve ter exatamente 3 participantes", nil))
		return
	}

	ids := make([]primitive.ObjectID, len(req.ParticipantIDs))
	for i, idStr := range req.ParticipantIDs {
		id, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			shared.RespondError(w, shared.NewBadRequestError("id de participante inválido: "+idStr, err))
			return
		}
		ids[i] = id
	}

	team, err := h.service.CreateManual(r.Context(), ids)
	if err != nil {
		switch {
		case errors.Is(err, ErrParticipantAlreadyInTeam):
			shared.RespondError(w, shared.NewConflictError(err.Error(), nil))
		case errors.Is(err, ErrNotEnoughSemesters):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao criar time manual", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusCreated, team)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	teams, err := h.service.List(r.Context())
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao listar times", err))
		return
	}
	shared.RespondJSON(w, http.StatusOK, teams)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	team, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		} else {
			shared.RespondError(w, shared.NewInternalServerError("erro ao buscar time", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, team)
}

type approveRequest struct {
	TeamName string `json:"teamName"`
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	var req approveRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}
	if req.TeamName == "" {
		shared.RespondError(w, shared.NewBadRequestError("nome do time é obrigatório para aprovação", nil))
		return
	}

	if err := h.service.Approve(r.Context(), id, req.TeamName); err != nil {
		switch {
		case errors.Is(err, ErrTeamNotFound):
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		case errors.Is(err, ErrInvalidTeamStatus):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao aprovar time", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "aprovado"})
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	if err := h.service.Reject(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrTeamNotFound):
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		case errors.Is(err, ErrInvalidTeamStatus):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao rejeitar time", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "rejeitado"})
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	if err := h.service.Cancel(r.Context(), id); err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		} else {
			shared.RespondError(w, shared.NewInternalServerError("erro ao cancelar time", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "cancelado"})
}
