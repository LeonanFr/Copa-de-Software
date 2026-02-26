package signup

import (
	"context"
	"errors"

	"copasoftware/internal/modules/participants"
	"copasoftware/internal/modules/teamnames"
	"copasoftware/internal/modules/teams"
	"copasoftware/internal/shared"
)

type Service struct {
	participantSvc *participants.Service
	teamSvc        *teams.Service
	teamNameSvc    *teamnames.Service
}

func NewService(
	participantSvc *participants.Service,
	teamSvc *teams.Service,
	teamNameSvc *teamnames.Service,
) *Service {
	return &Service{
		participantSvc: participantSvc,
		teamSvc:        teamSvc,
		teamNameSvc:    teamNameSvc,
	}
}

type IndividualInput struct {
	Matricula string
	Nome      string
	Semestre  int
}

type TeamInput struct {
	Participants []struct {
		Matricula string
		Nome      string
		Semestre  int
	}
}

func (s *Service) SignupIndividual(ctx context.Context, input IndividualInput) (*participants.Participant, error) {
	if !shared.IsValidMatricula(input.Matricula) {
		return nil, participants.ErrInvalidMatricula
	}
	if !shared.IsValidSemester(input.Semestre) {
		return nil, participants.ErrInvalidSemester
	}

	_, err := s.participantSvc.GetByMatricula(ctx, input.Matricula)
	if err == nil {
		return nil, participants.ErrParticipantAlreadyExists
	}
	if !errors.Is(err, participants.ErrParticipantNotFound) {
		return nil, err
	}

	inTeam, err := s.teamSvc.ExistsByMatriculaWithStatus(ctx, input.Matricula, []teams.TeamStatus{teams.TeamStatusPending, teams.TeamStatusApproved})
	if err != nil {
		return nil, err
	}
	if inTeam {
		return nil, errors.New("participante já está vinculado a um time")
	}

	return s.participantSvc.Create(ctx, input.Matricula, input.Nome, input.Semestre)
}

func (s *Service) SignupTeam(ctx context.Context, input TeamInput) (*teams.Team, error) {
	if len(input.Participants) != 3 {
		return nil, errors.New("time deve ter exatamente 3 participantes")
	}

	seen := make(map[string]struct{})
	participantData := make([]teams.ParticipantData, 3)

	for i, p := range input.Participants {
		if !shared.IsValidMatricula(p.Matricula) {
			return nil, participants.ErrInvalidMatricula
		}
		if !shared.IsValidSemester(p.Semestre) {
			return nil, participants.ErrInvalidSemester
		}
		if _, ok := seen[p.Matricula]; ok {
			return nil, errors.New("matrícula duplicada no time")
		}
		seen[p.Matricula] = struct{}{}

		_, err := s.participantSvc.GetByMatricula(ctx, p.Matricula)
		if err == nil {
			return nil, participants.ErrParticipantAlreadyExists
		}
		if !errors.Is(err, participants.ErrParticipantNotFound) {
			return nil, err
		}

		participantData[i] = teams.ParticipantData{
			Matricula: p.Matricula,
			Nome:      p.Nome,
			Semestre:  p.Semestre,
		}
	}

	team, err := s.teamSvc.CreatePending(ctx, participantData)
	if err != nil {
		return nil, err
	}
	return team, nil
}
