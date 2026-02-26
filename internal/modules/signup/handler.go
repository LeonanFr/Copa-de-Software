package signup

import (
	"copasoftware/internal/modules/teamnames"
	"copasoftware/internal/modules/teams"
	"errors"
	"log"
	"net/http"

	"copasoftware/internal/modules/participants"
	"copasoftware/internal/shared"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *Service
}

func NewHandler(router *mux.Router, service *Service) {
	h := &Handler{service: service}

	router.HandleFunc("/signup/individual", h.signupIndividual).Methods("POST")
	router.HandleFunc("/signup/team", h.signupTeam).Methods("POST")
}

func (h *Handler) signupIndividual(w http.ResponseWriter, r *http.Request) {
	var payload IndividualPayload
	if err := shared.DecodeAndValidate(r, &payload); err != nil {
		shared.RespondError(w, err)
		return
	}

	input := IndividualInput{
		Matricula: payload.Matricula,
		Nome:      payload.Nome,
		Semestre:  payload.Semestre,
	}

	p, err := h.service.SignupIndividual(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, participants.ErrInvalidSemester):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		case errors.Is(err, participants.ErrParticipantAlreadyExists):
			shared.RespondError(w, shared.NewConflictError(err.Error(), nil))
		default:
			shared.RespondError(w, shared.NewInternalServerError("erro na inscrição individual", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusCreated, p)
}

func (h *Handler) signupTeam(w http.ResponseWriter, r *http.Request) {
	var payload TeamPayload
	if err := shared.DecodeAndValidate(r, &payload); err != nil {
		shared.RespondError(w, err)
		return
	}

	if len(payload.Participants) == 0 {
		shared.RespondError(w, shared.NewBadRequestError("time deve ter pelo menos um participante", nil))
		return
	}
	if len(payload.Participants) != 3 {
		shared.RespondError(w, shared.NewBadRequestError("time deve ter exatamente 3 participantes", nil))
		return
	}

	input := TeamInput{}
	for _, part := range payload.Participants {
		input.Participants = append(input.Participants, struct {
			Matricula string
			Nome      string
			Semestre  int
		}{
			Matricula: part.Matricula,
			Nome:      part.Nome,
			Semestre:  part.Semestre,
		})
	}

	team, err := h.service.SignupTeam(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, participants.ErrInvalidSemester):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		case errors.Is(err, participants.ErrParticipantAlreadyExists):
			shared.RespondError(w, shared.NewConflictError(err.Error(), nil))
		case errors.Is(err, participants.ErrParticipantNotFound):
			shared.RespondError(w, shared.NewNotFoundError(err.Error(), nil))
		case errors.Is(err, teams.ErrParticipantAlreadyInTeam):
			shared.RespondError(w, shared.NewConflictError(err.Error(), nil))
		case errors.Is(err, teams.ErrNotEnoughSemesters):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		case errors.Is(err, participants.ErrInvalidMatricula):
			shared.RespondError(w, shared.NewBadRequestError(err.Error(), nil))
		case errors.Is(err, teamnames.ErrNoNamesAvailable):
			shared.RespondError(w, shared.NewConflictError(err.Error(), nil))
		default:
			log.Printf("Erro inesperado: %v", err)
			shared.RespondError(w, shared.NewInternalServerError("erro na inscrição em time", err))
		}
		return
	}
	shared.RespondJSON(w, http.StatusCreated, team)
}
