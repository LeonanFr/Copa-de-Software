package pendingteams

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrPendingTeamNotFound = errors.New("time pendente não encontrado")
	ErrInvalidStatus       = errors.New("status inválido para esta operação")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, teamName string, participants []ParticipantData) (*PendingTeam, error) {
	pt := &PendingTeam{
		TeamName:     teamName,
		Participants: participants,
	}
	if err := s.repo.Insert(ctx, pt); err != nil {
		return nil, err
	}
	return pt, nil
}

func (s *Service) GetByID(ctx context.Context, id primitive.ObjectID) (*PendingTeam, error) {
	pt, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pt == nil {
		return nil, ErrPendingTeamNotFound
	}
	return pt, nil
}

func (s *Service) MarkApproved(ctx context.Context, id primitive.ObjectID) error {
	pt, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if pt.Status != StatusPending {
		return ErrInvalidStatus
	}
	return s.repo.UpdateStatus(ctx, id, StatusApproved)
}

func (s *Service) MarkRejected(ctx context.Context, id primitive.ObjectID) error {
	pt, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if pt.Status != StatusPending {
		return ErrInvalidStatus
	}
	return s.repo.UpdateStatus(ctx, id, StatusRejected)
}
