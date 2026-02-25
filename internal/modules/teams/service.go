package teams

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"copasoftware/internal/modules/participants"
	"copasoftware/internal/modules/teamnames"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrTeamNotFound             = errors.New("time não encontrado")
	ErrInvalidTeamStatus        = errors.New("status inválido para esta operação")
	ErrParticipantAlreadyInTeam = errors.New("participante já está em um time ativo")
	ErrNotEnoughSemesters       = errors.New("time deve ter participantes de pelo menos 3 semestres diferentes")
)

type Service struct {
	repo           *Repository
	participantSvc *participants.Service
	teamNameSvc    *teamnames.Service
}

func NewService(repo *Repository, participantSvc *participants.Service, teamNameSvc *teamnames.Service) *Service {
	return &Service{
		repo:           repo,
		participantSvc: participantSvc,
		teamNameSvc:    teamNameSvc,
	}
}

func (s *Service) validateDifferentSemesters(ctx context.Context, participantIDs []primitive.ObjectID) error {
	semesters := make(map[int]bool)
	for _, pid := range participantIDs {
		p, err := s.participantSvc.GetByID(ctx, pid)
		if err != nil {
			return err
		}
		semesters[p.Semestre] = true
	}
	if len(semesters) < 3 {
		return ErrNotEnoughSemesters
	}
	return nil
}

func generateTeamCode(teamName string, totalSemesterSum int) string {
	firstLetter := "X"
	if len(teamName) > 0 {
		firstLetter = strings.ToUpper(string(teamName[0]))
	}

	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	hash := make([]byte, 4)
	for i := range hash {
		hash[i] = charset[rand.Intn(len(charset))]
	}

	return fmt.Sprintf("%s%d-%s", firstLetter, totalSemesterSum, string(hash))
}

func (s *Service) calculateSemesterSum(ctx context.Context, participantIDs []primitive.ObjectID) (int, error) {
	sum := 0
	for _, pid := range participantIDs {
		p, err := s.participantSvc.GetByID(ctx, pid)
		if err != nil {
			return 0, err
		}
		sum += p.Semestre
	}
	return sum, nil
}

func (s *Service) CreateManual(ctx context.Context, participantIDs []primitive.ObjectID) (*Team, error) {
	if len(participantIDs) != 3 {
		return nil, errors.New("time deve ter exatamente 3 participantes")
	}

	if err := s.validateDifferentSemesters(ctx, participantIDs); err != nil {
		return nil, err
	}

	for _, pid := range participantIDs {
		_, err := s.participantSvc.GetByID(ctx, pid)
		if err != nil {
			return nil, err
		}

		teams, err := s.repo.FindByParticipant(ctx, pid)
		if err != nil {
			return nil, err
		}
		for _, t := range teams {
			if t.Status == TeamStatusPending || t.Status == TeamStatusApproved {
				return nil, ErrParticipantAlreadyInTeam
			}
		}
	}

	team := &Team{
		Participants: participantIDs,
		Status:       TeamStatusPending,
		IsDraw:       false,
	}
	if err := s.repo.Insert(ctx, team); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *Service) CreateDraw(ctx context.Context, participantIDs []primitive.ObjectID) (*Team, error) {
	if len(participantIDs) != 3 {
		return nil, errors.New("time deve ter exatamente 3 participantes")
	}

	if err := s.validateDifferentSemesters(ctx, participantIDs); err != nil {
		return nil, err
	}

	for _, pid := range participantIDs {
		_, err := s.participantSvc.GetByID(ctx, pid)
		if err != nil {
			return nil, err
		}

		teams, err := s.repo.FindByParticipant(ctx, pid)
		if err != nil {
			return nil, err
		}
		for _, t := range teams {
			if t.Status == TeamStatusPending || t.Status == TeamStatusApproved {
				return nil, ErrParticipantAlreadyInTeam
			}
		}
	}

	name, err := s.teamNameSvc.ReserveOne(ctx, primitive.NilObjectID)
	if err != nil {
		return nil, err
	}

	semesterSum, err := s.calculateSemesterSum(ctx, participantIDs)
	if err != nil {
		return nil, err
	}
	code := generateTeamCode(name, semesterSum)

	team := &Team{
		Name:         name,
		Code:         code,
		Participants: participantIDs,
		Status:       TeamStatusApproved,
		IsDraw:       true,
	}

	if err := s.repo.Insert(ctx, team); err != nil {
		_ = s.teamNameSvc.ReleaseByTeam(ctx, team.ID)
		return nil, err
	}

	if err := s.teamNameSvc.ReleaseByTeam(ctx, team.ID); err != nil {
	}

	return team, nil
}

func (s *Service) Approve(ctx context.Context, teamID primitive.ObjectID, teamName string) error {
	team, err := s.repo.FindByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}
	if team.IsDraw {
		return ErrInvalidTeamStatus
	}
	if team.Status != TeamStatusPending {
		return ErrInvalidTeamStatus
	}

	semesterSum, err := s.calculateSemesterSum(ctx, team.Participants)
	if err != nil {
		return err
	}
	code := generateTeamCode(teamName, semesterSum)

	team.Status = TeamStatusApproved
	team.Name = teamName
	team.Code = code
	return s.repo.Update(ctx, team)
}

func (s *Service) Reject(ctx context.Context, teamID primitive.ObjectID) error {
	team, err := s.repo.FindByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}
	if team.IsDraw {
		return ErrInvalidTeamStatus
	}
	if team.Status != TeamStatusPending {
		return ErrInvalidTeamStatus
	}
	team.Status = TeamStatusRejected
	return s.repo.Update(ctx, team)
}

func (s *Service) Cancel(ctx context.Context, teamID primitive.ObjectID) error {
	team, err := s.repo.FindByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}
	if team.Status == TeamStatusCancelled {
		return nil
	}

	if team.IsDraw && team.Name != "" {
		if err := s.teamNameSvc.ReleaseByTeam(ctx, teamID); err != nil {
		}
	}

	team.Status = TeamStatusCancelled
	return s.repo.Update(ctx, team)
}

func (s *Service) GetByID(ctx context.Context, id primitive.ObjectID) (*Team, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTeamNotFound
	}
	return t, nil
}

func (s *Service) List(ctx context.Context) ([]Team, error) {
	return s.repo.FindAll(ctx)
}

func (s *Service) GetTeamsByParticipant(ctx context.Context, participantID primitive.ObjectID) ([]Team, error) {
	return s.repo.FindByParticipant(ctx, participantID)
}
