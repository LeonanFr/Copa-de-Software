package ranking

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"copasoftware/internal/shared"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler struct {
	service *Service
}

type EventRequest struct {
	TeamCode     string `json:"team_code"`
	TournamentID string `json:"tournament_id,omitempty"`
	Type         string `json:"type"`
	Matricula    string `json:"matricula,omitempty"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func RegisterPublicRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/ranking", h.getRanking).Methods("GET")
	router.HandleFunc("/ranking/team/{teamId}", h.getTeamRanking).Methods("GET")
	router.HandleFunc("/ranking/stream", h.RankingSSE).Methods("GET")
}

func RegisterAdminRoutes(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/ranking/score", h.addScore).Methods("POST")
	router.HandleFunc("/ranking/recalculate", h.recalculateAll).Methods("POST")
	router.HandleFunc("/ranking/scores/batch", h.batchAddScores).Methods("POST")
}

type batchAddScoresRequest struct {
	Scores []BatchScoreRequest `json:"scores"`
}

func (h *Handler) batchAddScores(w http.ResponseWriter, r *http.Request) {
	var req batchAddScoresRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}

	if len(req.Scores) == 0 {
		shared.RespondError(w, shared.NewBadRequestError("lote de pontuações vazio", nil))
		return
	}

	processed, err := h.service.AddScoresBatch(r.Context(), req.Scores)
	if err != nil {
		var appErr *shared.AppError
		if errors.As(err, &appErr) {
			shared.RespondError(w, appErr)
		}
		return
	}

	ranking, _ := h.service.GetFullRanking(r.Context())
	Manager.Broadcast(ranking)

	shared.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]int{
			"processed": processed,
		},
		"message": "Lote de pontuações registrado com sucesso.",
	})
}

func (h *Handler) getRanking(w http.ResponseWriter, r *http.Request) {
	ranking, err := h.service.GetFullRanking(r.Context())
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
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recálculo em background entrou em panic: %v", r)
			}
		}()

		ctx := context.Background()
		idsComFalha, err := h.service.RecalculateAll(ctx)
		if err != nil {
			log.Printf("Recálculo em background falhou: %v", err)
		} else if len(idsComFalha) > 0 {
			log.Printf("Recálculo concluído com falhas: %v", idsComFalha)
		} else {
			log.Printf("Recálculo concluído com sucesso")
		}
	}()

	shared.RespondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Recálculo iniciado em segundo plano",
	})
}

func (h *Handler) HandlePuzzleEvent(w http.ResponseWriter, r *http.Request) {
	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.RespondError(w, shared.NewBadRequestError("corpo inválido", err))
		return
	}
	if req.TeamCode == "" {
		shared.RespondError(w, shared.NewBadRequestError("team_code obrigatório", nil))
		return
	}
	err := h.service.AddPuzzleEvent(r.Context(), req.TeamCode, req.Matricula)
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao processar evento", err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleCaseEvent(w http.ResponseWriter, r *http.Request) {
	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.RespondError(w, shared.NewBadRequestError("corpo inválido", err))
		return
	}
	if req.TeamCode == "" {
		shared.RespondError(w, shared.NewBadRequestError("team_code obrigatório", nil))
		return
	}
	err := h.service.AddCaseEvent(r.Context(), req.TeamCode)
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao processar evento", err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleAlgorithmEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TournamentID string `json:"tournamentId"`
		TeamCode     string `json:"teamCode"`
		Value        int    `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.RespondError(w, shared.NewBadRequestError("corpo inválido", err))
		return
	}

	if err := h.service.AddAlgorithmEvent(r.Context(), req.TeamCode, req.TournamentID, req.Value); err != nil {
		var appErr *shared.AppError
		if errors.As(err, &appErr) {
			shared.RespondError(w, appErr)
			return
		}

		shared.RespondError(w, shared.NewInternalServerError("erro ao processar evento de algoritmo", err))
		return
	}

	shared.RespondJSON(w, http.StatusOK, map[string]string{
		"status": "evento de algoritmo processado",
	})
}
