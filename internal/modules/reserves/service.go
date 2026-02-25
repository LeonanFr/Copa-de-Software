package reserves

import (
	"context"
	"errors"

	"copasoftware/internal/modules/participants"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrParticipantNotFound  = errors.New("participante não encontrado")
	ErrAlreadyInReserve     = errors.New("participante já está na reserva")
	ErrReserveEntryNotFound = errors.New("entrada na reserva não encontrada")
)

type Service struct {
	repo           *Repository
	participantSvc *participants.Service
}

func NewService(repo *Repository, participantSvc *participants.Service) *Service {
	return &Service{
		repo:           repo,
		participantSvc: participantSvc,
	}
}

func (s *Service) AddToReserve(ctx context.Context, participantID primitive.ObjectID) error {
	p, err := s.participantSvc.GetByID(ctx, participantID)
	if err != nil {
		return err
	}

	existing, err := s.repo.FindByParticipant(ctx, participantID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrAlreadyInReserve
	}

	entry := &ReserveEntry{
		Participant: participantID,
		Semestre:    p.Semestre,
	}
	return s.repo.Insert(ctx, entry)
}

func (s *Service) ListReserves(ctx context.Context) ([]ReserveEntry, error) {
	return s.repo.FindAll(ctx)
}

func (s *Service) RemoveFromReserve(ctx context.Context, entryID primitive.ObjectID) error {
	return s.repo.Delete(ctx, entryID)
}

func (s *Service) RemoveByParticipant(ctx context.Context, participantID primitive.ObjectID) error {
	return s.repo.DeleteByParticipant(ctx, participantID)
}

func (s *Service) IsInReserve(ctx context.Context, participantID primitive.ObjectID) (bool, error) {
	entry, err := s.repo.FindByParticipant(ctx, participantID)
	if err != nil {
		return false, err
	}
	return entry != nil, nil
}
