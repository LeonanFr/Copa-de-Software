package ranking

import (
	"context"
	"copasoftware/internal/modules/teams"
	"errors"
	"log"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrTeamNotFound = errors.New("time não encontrado")
)

type Service struct {
	repo    *Repository
	teamSvc *teams.Service
}

func NewService(repo *Repository, teamSvc *teams.Service) *Service {
	return &Service{
		repo:    repo,
		teamSvc: teamSvc,
	}
}

func (s *Service) AddScore(ctx context.Context, teamID primitive.ObjectID, value int, origin ScoreOrigin, modality, description string) error {
	team, err := s.teamSvc.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team.Status != teams.TeamStatusApproved {
		return errors.New("apenas times aprovados podem receber pontuação")
	}

	entry := &ScoreEntry{
		TeamID:      teamID,
		Value:       value,
		Origin:      origin,
		Modality:    modality,
		Description: description,
	}
	if err := s.repo.InsertScore(ctx, entry); err != nil {
		return err
	}

	if err := s.recalculateTeamRanking(ctx, teamID); err != nil {
		return err
	}

	ranking, _ := s.GetFullRanking(context.Background())
	Manager.Broadcast(ranking)

	return nil
}

func (s *Service) GetTeamRanking(ctx context.Context, teamID primitive.ObjectID) (*TeamRanking, error) {
	tr, err := s.repo.FindRankingByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if tr == nil {
		return &TeamRanking{
			TeamID: teamID,
			Total:  0,
		}, nil
	}
	return tr, nil
}

func (s *Service) recalculateTeamRanking(ctx context.Context, teamID primitive.ObjectID) error {
	scores, err := s.repo.FindScoresByTeam(ctx, teamID)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}
	total := 0
	for _, sc := range scores {
		total += sc.Value
	}
	tr := &TeamRanking{
		TeamID: teamID,
		Total:  total,
	}
	return s.repo.UpsertRanking(ctx, tr)
}

func (s *Service) RecalculateAll(ctx context.Context) ([]string, error) {
	if err := s.repo.RecalculateAllRankings(ctx); err != nil {
		return nil, err
	}

	ranking, _ := s.GetFullRanking(ctx)
	Manager.Broadcast(ranking)
	return nil, nil
}

func (s *Service) InitializeTeam(ctx context.Context, teamID primitive.ObjectID) error {
	if err := s.recalculateTeamRanking(ctx, teamID); err != nil {
		log.Printf("InitializeTeam: erro ao recalcular time %s: %v", teamID.Hex(), err)
		return err
	}

	ranking, err := s.GetFullRanking(ctx)
	if err != nil {
		log.Printf("erro obtendo ranking completo: %v", err)
		return err
	}
	Manager.Broadcast(ranking)

	return nil
}

func (s *Service) GetFullRanking(ctx context.Context) ([]RankingEntry, error) {
	rankings, err := s.repo.FindAllRankings(ctx)
	if err != nil {
		return nil, err
	}
	if len(rankings) == 0 {
		return []RankingEntry{}, nil
	}

	teamIDs := make([]primitive.ObjectID, len(rankings))
	for i, r := range rankings {
		teamIDs[i] = r.TeamID
	}

	teamsList, err := s.teamSvc.FindByIDs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}

	teamMap := make(map[primitive.ObjectID]*teams.Team)
	for _, t := range teamsList {
		teamMap[t.ID] = t
	}

	entries := make([]RankingEntry, 0, len(rankings))
	for _, r := range rankings {
		team, ok := teamMap[r.TeamID]
		if !ok {
			log.Printf("time %s não encontrado no ranking", r.TeamID.Hex())
			continue
		}
		entries = append(entries, RankingEntry{
			TeamID:   r.TeamID,
			TeamName: team.Name,
			Total:    r.Total,
		})
	}
	return entries, nil
}

func (s *Service) AddPuzzleEvent(ctx context.Context, teamCode, matricula string) error {
	team, err := s.teamSvc.GetByCode(ctx, teamCode)
	if err != nil {
		return err
	}
	if team == nil {
		return errors.New("time não encontrado")
	}
	if team.Status != teams.TeamStatusApproved {
		return errors.New("time não aprovado")
	}
	return s.AddScore(ctx, team.ID, 10, OriginMatch, "puzzle", "Puzzle resolvido por "+matricula)
}

func (s *Service) AddCaseEvent(ctx context.Context, teamCode string) error {

	event := &ProcessedEvent{
		TeamCode: teamCode,
		Type:     "case",
	}
	err := s.repo.InsertProcessedEvent(ctx, event)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	}

	team, err := s.teamSvc.GetByCode(ctx, teamCode)
	if err != nil {
		return err
	}
	if team == nil {
		return errors.New("time não encontrado")
	}
	if team.Status != teams.TeamStatusApproved {
		return errors.New("time não aprovado")
	}

	return s.AddScore(ctx, team.ID, 60, OriginMatch, "case", "Torneio completo")
}

func (s *Service) AddAlgorithmEvent(ctx context.Context, teamCode string, tournamentID string) error {
	event := &ProcessedEvent{
		TeamCode: teamCode,
		Type:     "algorithm_" + tournamentID,
	}

	err := s.repo.InsertProcessedEvent(ctx, event)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	}

	team, err := s.teamSvc.GetByCode(ctx, teamCode)
	if err != nil {
		return err
	}
	if team == nil {
		return errors.New("time não encontrado")
	}
	if team.Status != teams.TeamStatusApproved {
		return errors.New("time não aprovado")
	}

	description := "Desafio de algoritmo aceito no torneio: " + tournamentID
	return s.AddScore(ctx, team.ID, 100, OriginMatch, "algorithm", description)
}

func (s *Service) DeleteByTeam(ctx context.Context, teamID primitive.ObjectID) error {
	return s.repo.DeleteRankingByTeam(ctx, teamID)
}
