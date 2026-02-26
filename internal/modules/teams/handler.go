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
	router.HandleFunc("/teams/participant/{matricula}", h.getByParticipantMatricula).Methods("GET")
}

func RegisterAdminRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/teams/{id}/approve", h.approve).Methods("POST")
	router.HandleFunc("/teams/{id}/reject", h.reject).Methods("POST")
	router.HandleFunc("/teams/{id}/cancel", h.cancel).Methods("POST")
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
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
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

func (h *Handler) getByParticipantMatricula(w http.ResponseWriter, r *http.Request) {
	matricula := mux.Vars(r)["matricula"]
	if matricula == "" {
		shared.RespondError(w, shared.NewBadRequestError("matrícula não fornecida", nil))
		return
	}

	teams, err := h.service.GetTeamsByMatricula(r.Context(), matricula)
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao buscar times do participante", err))
		return
	}

	shared.RespondJSON(w, http.StatusOK, teams)
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	if err := h.service.Approve(r.Context(), id); err != nil {
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
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
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
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
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
