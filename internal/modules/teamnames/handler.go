package teamnames

import (
	"errors"
	"net/http"

	"copasoftware/internal/shared"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *Service
}

func NewHandler(router *mux.Router, service *Service) {
	h := &Handler{service: service}

	router.HandleFunc("/team-names", h.addName).Methods("POST")
	router.HandleFunc("/team-names/available", h.listAvailable).Methods("GET")
}

type addNameRequest struct {
	Name string `json:"name"`
}

func (h *Handler) addName(w http.ResponseWriter, r *http.Request) {
	var req addNameRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}
	if req.Name == "" {
		shared.RespondError(w, shared.NewBadRequestError("nome não pode ser vazio", nil))
		return
	}

	err := h.service.AddName(r.Context(), req.Name)
	if err != nil {
		switch {
		case errors.Is(err, ErrNameAlreadyExists):
			shared.RespondError(w, shared.NewConflictError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro ao adicionar nome", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusCreated, map[string]string{"status": "nome adicionado"})
}

func (h *Handler) listAvailable(w http.ResponseWriter, r *http.Request) {
	names, err := h.service.ListAvailable(r.Context())
	if err != nil {
		shared.RespondError(w, shared.NewInternalServerError("erro ao listar nomes disponíveis", err))
		return
	}
	shared.RespondJSON(w, http.StatusOK, names)
}
