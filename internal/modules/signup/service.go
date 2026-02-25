package signup

import (
	"context"
	"errors"

	"copasoftware/internal/modules/participants"
	"copasoftware/internal/modules/teams"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	participantSvc *participants.Service
	teamSvc        *teams.Service
}

func NewService(participantSvc *participants.Service, teamSvc *teams.Service) *Service {
	return &Service{
		participantSvc: participantSvc,
		teamSvc:        teamSvc,
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
	p, err := s.participantSvc.Create(ctx, input.Matricula, input.Nome, input.Semestre)
	if err != nil {
		return nil, err
	}

	teamsList, err := s.teamSvc.GetTeamsByParticipant(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	for _, t := range teamsList {
		if t.Status == teams.TeamStatusPending || t.Status == teams.TeamStatusApproved {
			return nil, errors.New("participante já está em um time ativo")
		}
	}

	return p, nil
}

func (s *Service) SignupTeam(ctx context.Context, input TeamInput) (*teams.Team, error) {
	if len(input.Participants) < 3 {
		return nil, errors.New("time deve ter exatamente 3 participantes")
	}

	participantIDs := make([]primitive.ObjectID, 0, len(input.Participants))
	for _, part := range input.Participants {
		p, err := s.participantSvc.Create(ctx, part.Matricula, part.Nome, part.Semestre)
		if err != nil {
			return nil, err
		}
		participantIDs = append(participantIDs, p.ID)
	}

	return s.teamSvc.CreateManual(ctx, participantIDs)
}
