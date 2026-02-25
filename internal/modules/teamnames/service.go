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

func (s *Service) ReserveOne(ctx context.Context, teamID primitive.ObjectID) (string, error) {
	available, err := s.repo.FindAvailable(ctx)
	if err != nil {
		return "", err
	}
	if len(available) == 0 {
		return "", ErrNoNamesAvailable
	}

	rand.Seed(time.Now().UnixNano())
	chosen := available[rand.Intn(len(available))]

	if err := s.repo.MarkAsUsed(ctx, chosen.ID, teamID); err != nil {
		return "", err
	}
	return chosen.Name, nil
}

func (s *Service) ReleaseByTeam(ctx context.Context, teamID primitive.ObjectID) error {
	return s.repo.MarkAsAvailable(ctx, teamID)
}

func (s *Service) ListAvailable(ctx context.Context) ([]TeamName, error) {
	return s.repo.FindAvailable(ctx)
}
