package teamnames

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrNameAlreadyExists = errors.New("nome de time já existe na lista")
	ErrNoNamesAvailable  = errors.New("nenhum nome de time disponível")
	ErrNameNotFound      = errors.New("nome não encontrado")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) AddName(ctx context.Context, name string) error {
	existing, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrNameAlreadyExists
	}

	tn := &TeamName{
		Name: name,
		Used: false,
	}
	return s.repo.Insert(ctx, tn)
}

func (s *Service) ReserveOne(ctx context.Context) (primitive.ObjectID, string, error) {
	available, err := s.repo.FindAvailable(ctx)
	if err != nil {
		return primitive.NilObjectID, "", err
	}
	if len(available) == 0 {
		return primitive.NilObjectID, "", ErrNoNamesAvailable
	}

	rand.Seed(time.Now().UnixNano())
	chosen := available[rand.Intn(len(available))]

	if err := s.repo.MarkAsUsed(ctx, chosen.ID, primitive.NilObjectID); err != nil {
		return primitive.NilObjectID, "", err
	}
	return chosen.ID, chosen.Name, nil
}

func (s *Service) AssignToTeam(ctx context.Context, nameID primitive.ObjectID, teamID primitive.ObjectID) error {
	tn, err := s.repo.FindByID(ctx, nameID)
	if err != nil {
		return err
	}
	if tn == nil {
		return ErrNameNotFound
	}
	if !tn.Used {
		return errors.New("nome não está reservado")
	}
	if tn.UsedByTeam != nil && *tn.UsedByTeam != primitive.NilObjectID {
		return errors.New("nome já está associado a outro time")
	}
	return s.repo.AssignToTeam(ctx, nameID, teamID)
}

func (s *Service) ReleaseByTeam(ctx context.Context, teamID primitive.ObjectID) error {
	return s.repo.MarkAsAvailableByTeam(ctx, teamID)
}

func (s *Service) ReleaseByID(ctx context.Context, nameID primitive.ObjectID) error {
	return s.repo.MarkAsAvailableByID(ctx, nameID)
}

func (s *Service) ListAvailable(ctx context.Context) ([]TeamName, error) {
	return s.repo.FindAvailable(ctx)
}
