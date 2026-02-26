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

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrTeamNotFound             = errors.New("time não encontrado")
	ErrInvalidTeamStatus        = errors.New("status inválido para esta operação")
	ErrParticipantAlreadyInTeam = errors.New("participante já está em um time ativo")
	ErrNotEnoughSemesters       = errors.New("time deve ter participantes de pelo menos 2 semestres diferentes")
)

type Service struct {
	repo           *Repository
	participantSvc *participants.Service
	teamNameSvc    *teamnames.Service
}

func NewService(repo *Repository, participantSvc *participants.Service, teamNameSvc *teamnames.Service) *Service {
	return &Service{repo, participantSvc, teamNameSvc}
}

func validateDifferentSemesters(data []ParticipantData) error {
	seen := map[int]bool{}
	for _, p := range data {
		seen[p.Semestre] = true
	}
	if len(seen) < 2 {
		return ErrNotEnoughSemesters
	}
	return nil
}

func generateTeamCode(teamName string, semesterSum int) string {
	first := "X"
	if teamName != "" {
		first = strings.ToUpper(teamName[:1])
	}
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	hash := make([]byte, 4)
	for i := range hash {
		hash[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("%s%d-%s", first, semesterSum, string(hash))
}

func (s *Service) CreatePending(
	ctx context.Context,
	data []ParticipantData,
) (*Team, error) {

	if len(data) != 3 {
		return nil, errors.New("time deve ter exatamente 3 participantes")
	}

	if err := validateDifferentSemesters(data); err != nil {
		return nil, err
	}

	for _, p := range data {
		if !shared.IsValidMatricula(p.Matricula) {
			return nil, participants.ErrInvalidMatricula
		}
		if !shared.IsValidSemester(p.Semestre) {
			return nil, participants.ErrInvalidSemester
		}

		exists, err := s.repo.ExistsByMatriculaWithStatus(
			ctx,
			p.Matricula,
			[]TeamStatus{TeamStatusPending, TeamStatusApproved},
		)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrParticipantAlreadyInTeam
		}
	}

	nameID, name, err := s.teamNameSvc.ReserveOne(ctx)
	if err != nil {
		return nil, err
	}

	team := &Team{
		Name:            name,
		ParticipantData: data,
		Status:          TeamStatusPending,
	}

	if err := s.repo.Insert(ctx, team); err != nil {
		_ = s.teamNameSvc.ReleaseByID(ctx, nameID)
		return nil, err
	}

	_ = s.teamNameSvc.AssignToTeam(ctx, nameID, team.ID)
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
	if team.Status != TeamStatusPending {
		return ErrInvalidTeamStatus
	}

	var ids []primitive.ObjectID
	sum := 0

	for _, pd := range team.ParticipantData {
		p, err := s.participantSvc.Create(ctx, pd.Matricula, pd.Nome, pd.Semestre)
		if err != nil {
			for _, id := range ids {
				_ = s.participantSvc.Cancel(ctx, id)
			}
			return err
		}
		ids = append(ids, p.ID)
		sum += pd.Semestre
	}

	team.Code = generateTeamCode(team.Name, sum)
	team.Participants = ids
	team.ParticipantData = nil
	team.Status = TeamStatusApproved

	return s.repo.Update(ctx, team)
}

func (s *Service) Reject(ctx context.Context, id primitive.ObjectID) error {
	team, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}
	if team.Status != TeamStatusPending {
		return ErrInvalidTeamStatus
	}

	_ = s.teamNameSvc.ReleaseByTeam(ctx, id)
	team.Status = TeamStatusRejected
	return s.repo.Update(ctx, team)
}

func (s *Service) Cancel(ctx context.Context, id primitive.ObjectID) error {
	team, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	_ = s.teamNameSvc.ReleaseByTeam(ctx, id)
	team.Status = TeamStatusCancelled
	return s.repo.Update(ctx, team)
}

func (s *Service) ExistsByMatriculaWithStatus(
	ctx context.Context,
	matricula string,
	statuses []TeamStatus,
) (bool, error) {
	return s.repo.ExistsByMatriculaWithStatus(ctx, matricula, statuses)
}
