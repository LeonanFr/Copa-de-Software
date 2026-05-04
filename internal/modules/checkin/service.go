package checkin

import (
	"context"
	"errors"

	"copasoftware/internal/modules/participants"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrCheckinAlreadyExists = errors.New("participante já possui check-in")
	ErrCheckinNotFound      = errors.New("check-in não encontrado")
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

func (s *Service) CheckinParticipant(ctx context.Context, participantID primitive.ObjectID) (*Checkin, error) {

	p, err := s.participantSvc.GetByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.FindByParticipantID(ctx, participantID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrCheckinAlreadyExists
	}

	c := &Checkin{
		ParticipantID: &participantID,
		Nome:          p.Nome,
		Tipo:          TipoCompetidor,
	}

	if err := s.repo.Insert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) AdicionarOuvinte(ctx context.Context, nome string) (*Checkin, error) {
	c := &Checkin{
		Tipo: TipoOuvinte,
		Nome: nome,
	}
	if err := s.repo.Insert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) RemoverCheckin(ctx context.Context, id primitive.ObjectID) error {

	all, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, c := range all {
		if c.ID == id {
			found = true
			break
		}
	}
	if !found {
		return ErrCheckinNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) ListarCheckins(ctx context.Context) ([]Checkin, error) {
	return s.repo.FindAll(ctx)
}
