package pendingteams

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

func RegisterAdminRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/pending-teams/{id}/approve", h.approve).Methods("POST")
	router.HandleFunc("/pending-teams/{id}/reject", h.reject).Methods("POST")
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	if err := h.service.MarkApproved(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrPendingTeamNotFound):
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		case errors.Is(err, ErrInvalidStatus):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao aprovar time pendente", err))
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

	if err := h.service.MarkRejected(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrPendingTeamNotFound):
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		case errors.Is(err, ErrInvalidStatus):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao rejeitar time pendente", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "rejeitado"})
}
