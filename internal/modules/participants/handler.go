package participants

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

func NewHandler(router *mux.Router, service *Service) {
	h := &Handler{service: service}

	router.HandleFunc("/participants", h.list).Methods("GET")
	router.HandleFunc("/participants/{id}", h.getByID).Methods("GET")
	router.HandleFunc("/participants/matricula/{matricula}", h.getByMatricula).Methods("GET")
	router.HandleFunc("/participants/{id}", h.update).Methods("PUT")
	router.HandleFunc("/participants/{id}/cancel", h.cancel).Methods("POST")
}

type createRequest struct {
	Matricula string `json:"matricula"`
	Nome      string `json:"nome"`
	Semestre  int    `json:"semestre"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}

	p, err := h.service.Create(r.Context(), req.Matricula, req.Nome, req.Semestre)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidSemester):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		case errors.Is(err, ErrParticipantAlreadyExists):
			shared.RespondError(w, shared.NewConflictError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao criar participante", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusCreated, p)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	participants, err := h.service.List(r.Context())
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao listar participantes", err))
		return
	}
	shared.RespondJSON(w, http.StatusOK, participants)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	p, err := h.service.GetByID(r.Context(), id)
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

func (h *Handler) getByMatricula(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matricula := vars["matricula"]

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
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	var req updateRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}

	p, err := h.service.Update(r.Context(), id, req.Nome, req.Semestre)
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
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("id inválido", err))
		return
	}

	if err := h.service.Cancel(r.Context(), id); err != nil {
		if errors.Is(err, ErrParticipantNotFound) {
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		} else {
			shared.RespondError(w, shared.NewInternalServerError("erro ao cancelar participante", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusOK, map[string]string{"status": "cancelado"})
}
