package ranking

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
	router.HandleFunc("/ranking", h.getRanking).Methods("GET")
	router.HandleFunc("/ranking/team/{teamId}", h.getTeamRanking).Methods("GET")
}

func RegisterAdminRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/ranking/score", h.addScore).Methods("POST")
	router.HandleFunc("/ranking/recalculate", h.recalculateAll).Methods("POST")
}

func (h *Handler) getRanking(w http.ResponseWriter, r *http.Request) {
	ranking, err := h.service.GetRanking(r.Context())
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao obter ranking", err))
		return
	}
	shared.RespondJSON(w, http.StatusOK, ranking)
}

func (h *Handler) getTeamRanking(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID, err := primitive.ObjectIDFromHex(vars["teamId"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id do time inválido", err))
		return
	}

	tr, err := h.service.GetTeamRanking(r.Context(), teamID)
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		} else {
			shared.RespondError(w, shared.NewInternalServerError("erro ao obter ranking do time", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, tr)
}

type addScoreRequest struct {
	TeamID      string `json:"teamId"`
	Value       int    `json:"value"`
	Origin      string `json:"origin"`
	Modality    string `json:"modality,omitempty"`
	Description string `json:"description"`
}

func (h *Handler) addScore(w http.ResponseWriter, r *http.Request) {
	var req addScoreRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}

	teamID, err := primitive.ObjectIDFromHex(req.TeamID)
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id do time inválido", err))
		return
	}

	origin := ScoreOrigin(req.Origin)
	if origin != OriginMatch && origin != OriginPenalty && origin != OriginBonus {
		shared.RespondError(w, shared.NewBadRequestError("origem inválida", nil))
		return
	}

	if err := h.service.AddScore(r.Context(), teamID, req.Value, origin, req.Modality, req.Description); err != nil {
		switch {
		case errors.Is(err, ErrTeamNotFound):
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao adicionar pontuação", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusCreated, map[string]string{"status": "pontuação adicionada"})
}

func (h *Handler) recalculateAll(w http.ResponseWriter, r *http.Request) {
	if err := h.service.RecalculateAll(r.Context()); err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao recalcular ranking", err))
		return
	}
	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "ranking recalculado"})
}
