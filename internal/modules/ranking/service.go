package ranking

import (
	"context"
	"copasoftware/internal/modules/teams"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
	if err != nil {
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

func (s *Service) RecalculateAll(ctx context.Context) error {
	teamsList, err := s.teamSvc.List(ctx, nil)
	if err != nil {
		return err
	}
	for _, team := range teamsList {
		if team.Status == teams.TeamStatusApproved {
			if err := s.recalculateTeamRanking(ctx, team.ID); err != nil {
				return err
			}
		}
	}

	ranking, _ := s.GetFullRanking(context.Background())
	Manager.Broadcast(ranking)

	return nil
}

func (s *Service) InitializeTeam(ctx context.Context, teamID primitive.ObjectID) error {
	exists, err := s.repo.FindRankingByTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if exists != nil {
		return nil
	}
	tr := &TeamRanking{
		TeamID: teamID,
		Total:  0,
	}
	if err := s.repo.UpsertRanking(ctx, tr); err != nil {
		return err
	}

	ranking, _ := s.GetFullRanking(context.Background())
	Manager.Broadcast(ranking)

	return nil
}

func (s *Service) GetFullRanking(ctx context.Context) ([]RankingEntry, error) {
	rankings, err := s.repo.FindAllRankings(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]RankingEntry, 0, len(rankings))
	for _, r := range rankings {
		team, err := s.teamSvc.GetByID(ctx, r.TeamID)
		if err != nil {
			return nil, err
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
	return s.AddScore(ctx, team.ID, 60, OriginMatch, "case", "Caso completo")
}

func (s *Service) DeleteByTeam(ctx context.Context, teamID primitive.ObjectID) error {
	return s.repo.DeleteRankingByTeam(ctx, teamID)
}
