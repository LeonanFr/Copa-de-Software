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

func NewService(participantSvc *participants.Service, teamSvc *teams.Service, teamNameSvc *teamnames.Service) *Service {
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
	if len(input.Participants) != 3 {
		return nil, errors.New("time deve ter exatamente 3 participantes")
	}

	for _, p := range input.Participants {
		if !shared.IsValidMatricula(p.Matricula) {
			return nil, participants.ErrInvalidMatricula
		}
		if !shared.IsValidSemester(p.Semestre) {
			return nil, participants.ErrInvalidSemester
		}
		existing, _ := s.participantSvc.GetByMatricula(ctx, p.Matricula)
		if existing != nil {
			return nil, participants.ErrParticipantAlreadyExists
		}
	}

	nameID, name, err := s.teamNameSvc.ReserveOne(ctx)
	if err != nil {
		if errors.Is(err, teamnames.ErrNoNamesAvailable) {
			return nil, errors.New("não há nomes de times disponíveis no momento")
		}
		return nil, err
	}

	participantData := make([]teams.ParticipantData, len(input.Participants))
	for i, p := range input.Participants {
		participantData[i] = teams.ParticipantData{
			Matricula: p.Matricula,
			Nome:      p.Nome,
			Semestre:  p.Semestre,
		}
	}

	team := &teams.Team{
		Name:            name,
		ParticipantData: participantData,
		Status:          teams.TeamStatusPending,
		IsDraw:          false,
	}

	if err := s.teamSvc.CreatePending(ctx, team); err != nil {
		_ = s.teamNameSvc.ReleaseByID(ctx, nameID)
		return nil, err
	}

	if err := s.teamNameSvc.AssignToTeam(ctx, nameID, team.ID); err != nil {
		_ = s.teamSvc.Delete(ctx, team.ID)
		_ = s.teamNameSvc.ReleaseByID(ctx, nameID)
		return nil, err
	}

	return team, nil
}
