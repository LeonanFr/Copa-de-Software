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
	"copasoftware/internal/shared"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrTeamNotFound             = errors.New("time não encontrado")
	ErrInvalidTeamStatus        = errors.New("status inválido para esta operação")
	ErrParticipantAlreadyInTeam = errors.New("participante já está em um time ativo")
	ErrNotEnoughSemesters       = errors.New("time deve ter participantes de pelo menos 3 semestres diferentes")
)

type RankingInitializer interface {
	InitializeTeam(ctx context.Context, teamID primitive.ObjectID) error
}

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

func (s *Service) CreatePending(ctx context.Context, team *Team) error {
	if team.Status != TeamStatusPending {
		return ErrInvalidTeamStatus
	}
	return s.repo.Insert(ctx, team)
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

	nameID, name, err := s.teamNameSvc.ReserveOne(ctx)
	if err != nil {
		return nil, err
	}

	semesterSum, err := s.calculateSemesterSum(ctx, participantIDs)
	if err != nil {
		_ = s.teamNameSvc.ReleaseByID(ctx, nameID)
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
		_ = s.teamNameSvc.ReleaseByID(ctx, nameID)
		return nil, err
	}

	if err := s.teamNameSvc.AssignToTeam(ctx, nameID, team.ID); err != nil {
		_ = s.teamNameSvc.ReleaseByID(ctx, nameID)
		_ = s.Delete(ctx, team.ID)
		return nil, err
	}

	return team, nil
}

func (s *Service) Approve(ctx context.Context, teamID primitive.ObjectID) error {
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
	if len(team.ParticipantData) == 0 {
		return errors.New("time pendente não contém dados dos participantes")
	}
	if len(team.ParticipantData) != 3 {
		return errors.New("dados dos participantes inválidos")
	}

	participantIDs := make([]primitive.ObjectID, 0, 3)
	for _, pd := range team.ParticipantData {
		existing, _ := s.participantSvc.GetByMatricula(ctx, pd.Matricula)
		if existing != nil {
			return participants.ErrParticipantAlreadyExists
		}
		if !shared.IsValidSemester(pd.Semestre) {
			return participants.ErrInvalidSemester
		}
	}

	for _, pd := range team.ParticipantData {
		p, err := s.participantSvc.Create(ctx, pd.Matricula, pd.Nome, pd.Semestre)
		if err != nil {
			for _, pid := range participantIDs {
				_ = s.participantSvc.Cancel(ctx, pid)
			}
			return err
		}
		participantIDs = append(participantIDs, p.ID)
	}

	if err := s.validateDifferentSemesters(ctx, participantIDs); err != nil {
		for _, pid := range participantIDs {
			_ = s.participantSvc.Cancel(ctx, pid)
		}
		return err
	}

	semesterSum, err := s.calculateSemesterSum(ctx, participantIDs)
	if err != nil {
		for _, pid := range participantIDs {
			_ = s.participantSvc.Cancel(ctx, pid)
		}
		return err
	}
	code := generateTeamCode(team.Name, semesterSum)

	team.Code = code
	team.Participants = participantIDs
	team.ParticipantData = nil
	team.Status = TeamStatusApproved

	if err := s.repo.Update(ctx, team); err != nil {
		for _, pid := range participantIDs {
			_ = s.participantSvc.Cancel(ctx, pid)
		}
		return err
	}
	return nil
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

	if !team.IsDraw && team.Name != "" {
		_ = s.teamNameSvc.ReleaseByTeam(ctx, teamID)
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
		_ = s.teamNameSvc.ReleaseByTeam(ctx, teamID)
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

func (s *Service) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.repo.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
