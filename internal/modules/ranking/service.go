package ranking

import (
	"context"
	"copasoftware/internal/modules/teams"
	"copasoftware/internal/shared"
	"errors"
	"log"
	"strconv"

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

type BatchScoreRequest struct {
	TeamID      string `json:"teamId"`
	Value       int    `json:"value"`
	Origin      string `json:"origin"`
	Modality    string `json:"modality,omitempty"`
	Description string `json:"description"`
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

func (s *Service) AddScoresBatch(ctx context.Context, reqs []BatchScoreRequest) (int, error) {
	if len(reqs) == 0 {
		return 0, shared.NewBadRequestError("lote de pontuações vazio", nil)
	}

	session, err := s.repo.StartSession()
	if err != nil {
		return 0, shared.NewInternalServerError("erro ao iniciar sessão", err)
	}
	defer session.EndSession(ctx)

	inserted := 0
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		affectedTeams := make(map[primitive.ObjectID]struct{})

		for i, req := range reqs {
			if req.TeamID == "" {
				return nil, shared.NewBadRequestError("teamId é obrigatório no item "+strconv.Itoa(i), nil)
			}
			if req.Description == "" {
				return nil, shared.NewBadRequestError("description obrigatório no item "+strconv.Itoa(i), nil)
			}
			origin := ScoreOrigin(req.Origin)
			if origin != OriginMatch && origin != OriginPenalty && origin != OriginBonus {
				return nil, shared.NewBadRequestError("origem inválida no item "+strconv.Itoa(i)+": "+req.Origin, nil)
			}

			teamID, err := primitive.ObjectIDFromHex(req.TeamID)
			if err != nil {
				return nil, shared.NewBadRequestError("teamId inválido no item "+strconv.Itoa(i), err)
			}

			team, err := s.teamSvc.GetByID(sessCtx, teamID)
			if err != nil {
				if errors.Is(err, teams.ErrTeamNotFound) {
					return nil, shared.NewNotFoundError("Time com ID "+req.TeamID+" não encontrado", nil)
				}
				return nil, shared.NewInternalServerError("erro ao buscar time", err)
			}
			if team.Status != teams.TeamStatusApproved {
				return nil, shared.NewBadRequestError("Time com ID "+req.TeamID+" não está aprovado", nil)
			}

			entry := &ScoreEntry{
				TeamID:      teamID,
				Value:       req.Value,
				Origin:      origin,
				Modality:    req.Modality,
				Description: req.Description,
			}
			if err := s.repo.InsertScore(sessCtx, entry); err != nil {
				return nil, shared.NewInternalServerError("erro ao inserir pontuação", err)
			}
			inserted++
			affectedTeams[teamID] = struct{}{}
		}

		for teamID := range affectedTeams {
			if err := s.recalculateTeamRanking(sessCtx, teamID); err != nil {
				return nil, shared.NewInternalServerError("erro ao recalcular ranking", err)
			}
		}
		return nil, nil
	})

	if err != nil {
		return 0, err
	}
	return inserted, nil
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
	teamIDs, err := s.teamSvc.GetApprovedTeamIDs(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.repo.RecalculateAllRankings(ctx, teamIDs); err != nil {
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

func (s *Service) AddAlgorithmEvent(ctx context.Context, teamCode string, tournamentID string, value int) error {
	if teamCode == "" {
		return shared.NewBadRequestError("teamCode obrigatório", nil)
	}

	if tournamentID == "" {
		return shared.NewBadRequestError("tournamentId obrigatório", nil)
	}

	if value <= 0 {
		return shared.NewBadRequestError("value deve ser maior que zero", nil)
	}

	session, err := s.repo.StartSession()
	if err != nil {
		return shared.NewInternalServerError("erro ao iniciar sessão", err)
	}
	defer session.EndSession(ctx)

	var shouldBroadcast bool

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		team, err := s.teamSvc.GetByCode(sessCtx, teamCode)
		if err != nil {
			return nil, err
		}

		if team == nil {
			return nil, shared.NewNotFoundError("time não encontrado", nil)
		}

		if team.Status != teams.TeamStatusApproved {
			return nil, shared.NewBadRequestError("time não aprovado", nil)
		}

		event := &ProcessedEvent{
			TeamCode: teamCode,
			Type:     "algorithm:" + tournamentID,
		}

		created, err := s.repo.TryMarkProcessedEvent(sessCtx, event)
		if err != nil {
			return nil, shared.NewInternalServerError("erro ao registrar evento processado", err)
		}

		if !created {
			return nil, nil
		}

		entry := &ScoreEntry{
			TeamID:      team.ID,
			Value:       value,
			Origin:      OriginMatch,
			Modality:    "algorithm",
			Description: "Desafio de algoritmo aceito: " + tournamentID,
		}

		if err := s.repo.InsertScore(sessCtx, entry); err != nil {
			return nil, shared.NewInternalServerError("erro ao inserir pontuação", err)
		}

		if err := s.recalculateTeamRanking(sessCtx, team.ID); err != nil {
			return nil, shared.NewInternalServerError("erro ao recalcular ranking", err)
		}

		shouldBroadcast = true
		return nil, nil
	})

	if err != nil {
		return err
	}

	if shouldBroadcast {
		ranking, err := s.GetFullRanking(ctx)
		if err != nil {
			return shared.NewInternalServerError("erro ao obter ranking atualizado", err)
		}
		Manager.Broadcast(ranking)
	}

	return nil
}

func (s *Service) DeleteByTeam(ctx context.Context, teamID primitive.ObjectID) error {
	return s.repo.DeleteRankingByTeam(ctx, teamID)
}
