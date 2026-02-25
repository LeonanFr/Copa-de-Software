package auth

import (
	"net/http"

	"copasoftware/internal/shared"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *Service
}

func NewHandler(router *mux.Router, service *Service) {
	h := &Handler{service: service}
	router.HandleFunc("/admin/login", h.login).Methods("POST")
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := shared.DecodeAndValidate(r, &req); err != nil {
		shared.RespondError(w, err)
		return
	}

	token, err := h.service.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		shared.RespondError(w, shared.NewBadRequestError("credenciais inválidas", err))
		return
	}

	shared.RespondJSON(w, http.StatusOK, loginResponse{Token: token})
}
