package ranking

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"copasoftware/internal/shared"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JobStatus struct {
	Done      bool      `json:"done"`
	Error     string    `json:"error,omitempty"`
	Falhas    []string  `json:"falhas,omitempty"`
	StartedAt time.Time `json:"startedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var (
	jobs   = map[string]*JobStatus{}
	jobsMu sync.RWMutex
)

func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			jobsMu.Lock()
			for id, job := range jobs {
				if time.Since(job.UpdatedAt) > 30*time.Minute {
					delete(jobs, id)
				}
			}
			jobsMu.Unlock()
		}
	}()
}

type Handler struct {
	service *Service
}

type EventRequest struct {
	TeamCode  string `json:"team_code"`
	Type      string `json:"type"`
	Matricula string `json:"matricula,omitempty"`
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
	router.HandleFunc("/ranking/recalculate/status/{jobId}", h.getRecalculationStatus).Methods("GET")
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
	jobID := uuid.New().String()

	jobsMu.Lock()
	jobs[jobID] = &JobStatus{
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	jobsMu.Unlock()

	go func() {
		ctx := context.Background()
		idsComFalha, err := h.service.RecalculateAll(ctx)

		jobsMu.Lock()
		defer jobsMu.Unlock()
		job, exists := jobs[jobID]
		if !exists {
			return
		}
		if err != nil {
			job.Error = err.Error()
		} else {
			job.Falhas = idsComFalha
		}
		job.Done = true
		job.UpdatedAt = time.Now()
	}()

	shared.RespondJSON(w, http.StatusAccepted, map[string]string{"jobId": jobID})
}

func (h *Handler) getRecalculationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	jobsMu.RLock()
	status, exists := jobs[jobID]
	jobsMu.RUnlock()

	if !exists {
		shared.RespondError(w, shared.NewNotFoundError("job não encontrado", nil))
		return
	}

	shared.RespondJSON(w, http.StatusOK, status)
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
