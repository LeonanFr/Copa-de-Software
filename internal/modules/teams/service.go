package teams

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"copasoftware/internal/modules/participants"
	"copasoftware/internal/modules/teamnames"
	"copasoftware/internal/shared"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RankingService interface {
	InitializeTeam(ctx context.Context, teamID primitive.ObjectID) error
	DeleteByTeam(ctx context.Context, teamID primitive.ObjectID) error
}

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
	rankingSvc     RankingService
}

func NewService(repo *Repository, participantSvc *participants.Service, teamNameSvc *teamnames.Service) *Service {
	return &Service{
		repo:           repo,
		participantSvc: participantSvc,
		teamNameSvc:    teamNameSvc,
	}
}

func (s *Service) SetRankingSvc(rankingSvc RankingService) {
	s.rankingSvc = rankingSvc
}
func (s *Service) List(ctx context.Context, statuses []TeamStatus) ([]*Team, error) {
	var teams []*Team
	var err error
	if len(statuses) > 0 {
		teams, err = s.repo.FindByStatuses(ctx, statuses)
	} else {
		teams, err = s.repo.List(ctx)
	}
	if err != nil {
		return nil, err
	}
	return s.enrichTeams(ctx, teams)
}

func (s *Service) GetTeamsByParticipant(ctx context.Context, participantID primitive.ObjectID, matricula string) ([]*Team, error) {
	teamsByID, err := s.repo.FindByParticipantID(ctx, participantID, []TeamStatus{TeamStatusPending, TeamStatusApproved})
	if err != nil {
		return nil, err
	}
	teamsByMatricula, err := s.repo.FindByParticipantMatricula(ctx, matricula, []TeamStatus{TeamStatusPending})
	if err != nil {
		return nil, err
	}
	seen := make(map[primitive.ObjectID]bool)
	var result []*Team
	for _, t := range append(teamsByID, teamsByMatricula...) {
		if !seen[t.ID] {
			seen[t.ID] = true
			result = append(result, t)
		}
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id primitive.ObjectID) (*Team, error) {
	team, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrTeamNotFound
	}
	enriched, err := s.enrichTeams(ctx, []*Team{team})
	if err != nil {
		return nil, err
	}
	return enriched[0], nil
}

func (s *Service) GetTeamsByMatricula(ctx context.Context, matricula string) ([]*Team, error) {
	pendingTeams, err := s.repo.FindByParticipantMatricula(ctx, matricula, []TeamStatus{TeamStatusPending})
	if err != nil {
		return nil, err
	}
	p, err := s.participantSvc.GetByMatricula(ctx, matricula)
	if err != nil && !errors.Is(err, participants.ErrParticipantNotFound) {
		return nil, err
	}
	var approvedTeams []*Team
	if p != nil {
		approvedTeams, err = s.repo.FindByParticipantID(ctx, p.ID, []TeamStatus{TeamStatusApproved})
		if err != nil {
			return nil, err
		}
	}
	seen := make(map[primitive.ObjectID]bool)
	var result []*Team
	for _, t := range append(pendingTeams, approvedTeams...) {
		if !seen[t.ID] {
			seen[t.ID] = true
			result = append(result, t)
		}
	}
	return result, nil
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

func (s *Service) CreatePending(ctx context.Context, data []ParticipantData) (*Team, error) {
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
		exists, err := s.ExistsByMatriculaWithStatus(ctx, p.Matricula, []TeamStatus{TeamStatusPending, TeamStatusApproved})
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

func (s *Service) CreateManual(ctx context.Context, participantIDs []primitive.ObjectID) (*Team, error) {
	if len(participantIDs) != 3 {
		return nil, errors.New("time deve ter exatamente 3 participantes")
	}

	participantsList := make([]*participants.Participant, 3)
	for i, id := range participantIDs {
		p, err := s.participantSvc.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		participantsList[i] = p
	}

	for _, p := range participantsList {
		exists, err := s.ExistsByMatriculaWithStatus(ctx, p.Matricula, []TeamStatus{TeamStatusPending, TeamStatusApproved})
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
		Name:         name,
		Participants: participantIDs,
		Status:       TeamStatusPending,
		IsDraw:       false,
	}

	if err := s.repo.Insert(ctx, team); err != nil {
		_ = s.teamNameSvc.ReleaseByID(ctx, nameID)
		return nil, err
	}

	_ = s.teamNameSvc.AssignToTeam(ctx, nameID, team.ID)
	return team, nil
}

func (s *Service) CreateDraw(ctx context.Context, participantIDs []primitive.ObjectID) (*Team, error) {
	if len(participantIDs) != 3 {
		return nil, errors.New("time deve ter exatamente 3 participantes")
	}
	participantsList := make([]*participants.Participant, 3)
	for i, id := range participantIDs {
		p, err := s.participantSvc.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		participantsList[i] = p
	}
	for _, p := range participantsList {
		exists, err := s.ExistsByMatriculaWithStatus(ctx, p.Matricula, []TeamStatus{TeamStatusApproved})
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
	sum := 0
	for _, p := range participantsList {
		sum += p.Semestre
	}
	code := generateTeamCode(name, sum)
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
	_ = s.teamNameSvc.AssignToTeam(ctx, nameID, team.ID)

	if s.rankingSvc != nil {
		if err := s.rankingSvc.InitializeTeam(ctx, team.ID); err != nil {
			log.Printf("ERRO ao inicializar ranking para time %s: %v", team.ID.Hex(), err)
		}
	}

	return team, nil
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
	if len(team.ParticipantData) > 0 {
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
		team.Participants = ids
		team.ParticipantData = nil
		team.Code = generateTeamCode(team.Name, sum)
	} else if len(team.Participants) > 0 {
		for _, pid := range team.Participants {
			p, err := s.participantSvc.GetByID(ctx, pid)
			if err != nil {
				return err
			}
			sum += p.Semestre
		}
		team.Code = generateTeamCode(team.Name, sum)
	} else {
		return errors.New("time sem participantes")
	}
	team.Status = TeamStatusApproved

	if err := s.repo.Update(ctx, team); err != nil {
		return err
	}

	if s.rankingSvc != nil {
		if err := s.rankingSvc.InitializeTeam(ctx, team.ID); err != nil {
			log.Printf("ERRO ao inicializar ranking para time %s: %v", team.ID.Hex(), err)
		}
	}
	return nil
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

	if err := s.repo.Update(ctx, team); err != nil {
		return err
	}

	if s.rankingSvc != nil {
		_ = s.rankingSvc.DeleteByTeam(ctx, id)
	}
	return nil
}

func (s *Service) ExistsByMatriculaWithStatus(ctx context.Context, matricula string, statuses []TeamStatus) (bool, error) {
	exists, err := s.repo.ExistsByMatriculaInParticipantData(ctx, matricula, statuses)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	p, err := s.participantSvc.GetByMatricula(ctx, matricula)
	if err != nil {
		if errors.Is(err, participants.ErrParticipantNotFound) {
			return false, nil
		}
		return false, err
	}
	return s.repo.ExistsByParticipantID(ctx, p.ID, statuses)
}

func (s *Service) ExistsByMatriculaWithStatusStrings(ctx context.Context, matricula string, statuses []string) (bool, error) {
	var ts []TeamStatus
	for _, st := range statuses {
		ts = append(ts, TeamStatus(st))
	}
	return s.ExistsByMatriculaWithStatus(ctx, matricula, ts)
}

func (s *Service) enrichTeams(ctx context.Context, teams []*Team) ([]*Team, error) {

	var allIDs []primitive.ObjectID
	for _, team := range teams {
		allIDs = append(allIDs, team.Participants...)
	}
	if len(allIDs) == 0 {
		return teams, nil
	}

	participantsList, err := s.participantSvc.FindByIDs(ctx, allIDs)
	if err != nil {
		return nil, err
	}

	participantMap := make(map[primitive.ObjectID]ParticipantData)
	for _, p := range participantsList {
		participantMap[p.ID] = ParticipantData{
			Matricula: p.Matricula,
			Nome:      p.Nome,
			Semestre:  p.Semestre,
		}
	}

	for _, team := range teams {
		if len(team.Participants) == 0 {
			continue
		}
		data := make([]ParticipantData, 0, len(team.Participants))
		for _, pid := range team.Participants {
			if pd, ok := participantMap[pid]; ok {
				data = append(data, pd)
			}
		}
		team.ParticipantData = data
	}
	return teams, nil
}

func (s *Service) GetByCode(ctx context.Context, code string) (*Team, error) {
	team, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrTeamNotFound
	}
	enriched, err := s.enrichTeams(ctx, []*Team{team})
	if err != nil {
		return nil, err
	}
	return enriched[0], nil
}

func (s *Service) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*Team, error) {
	if len(ids) == 0 {
		return []*Team{}, nil
	}
	filter := bson.M{"_id": bson.M{"$in": ids}}
	cursor, err := s.repo.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var teams []*Team
	if err = cursor.All(ctx, &teams); err != nil {
		return nil, err
	}
	return teams, nil
}
