package participants

import (
	"context"
	"errors"

	"copasoftware/internal/shared"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrParticipantNotFound      = errors.New("participante não encontrado")
	ErrParticipantAlreadyExists = errors.New("matrícula já cadastrada")
	ErrInvalidSemester          = errors.New("semestre deve ser entre 1 e 8")
	ErrInvalidMatricula         = errors.New("matrícula deve ter pelo menos 6 dígitos numéricos")
	ErrParticipantInActiveTeam  = errors.New("participante está em um time ativo e não pode ser cancelado")
)

type TeamChecker interface {
	ExistsByMatriculaWithStatusStrings(ctx context.Context, matricula string, statuses []string) (bool, error)
}
type ReserveRemover interface {
	RemoveByParticipant(ctx context.Context, participantID primitive.ObjectID) error
}

type Service struct {
	repo           *Repository
	teamChecker    TeamChecker
	reserveRemover ReserveRemover
}

func (s *Service) SetReserveRemover(remover ReserveRemover) {
	s.reserveRemover = remover
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetTeamChecker(checker TeamChecker) {
	s.teamChecker = checker
}

func (s *Service) Create(ctx context.Context, matricula, nome string, semestre int) (*Participant, error) {
	if !shared.IsValidSemester(semestre) {
		return nil, ErrInvalidSemester
	}
	if !shared.IsValidMatricula(matricula) {
		return nil, ErrInvalidMatricula
	}
	existing, err := s.repo.FindByMatricula(ctx, matricula)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrParticipantAlreadyExists
	}

	p := &Participant{
		Matricula: matricula,
		Nome:      nome,
		Semestre:  semestre,
	}

	if err := s.repo.Insert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetByID(ctx context.Context, id primitive.ObjectID) (*Participant, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrParticipantNotFound
	}
	return p, nil
}

func (s *Service) GetByMatricula(ctx context.Context, matricula string) (*Participant, error) {
	p, err := s.repo.FindByMatricula(ctx, matricula)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrParticipantNotFound
	}
	return p, nil
}

func (s *Service) List(ctx context.Context) ([]Participant, error) {
	return s.repo.FindAll(ctx)
}

func (s *Service) Update(ctx context.Context, id primitive.ObjectID, nome string, semestre int) (*Participant, error) {
	if !shared.IsValidSemester(semestre) {
		return nil, ErrInvalidSemester
	}

	p, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	p.Nome = nome
	p.Semestre = semestre

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) UpdateByMatricula(ctx context.Context, matricula string, nome string, semestre int) (*Participant, error) {
	p, err := s.GetByMatricula(ctx, matricula)
	if err != nil {
		return nil, err
	}
	return s.Update(ctx, p.ID, nome, semestre)
}

func (s *Service) Cancel(ctx context.Context, id primitive.ObjectID) error {
	p, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if s.teamChecker != nil {
		inTeam, err := s.teamChecker.ExistsByMatriculaWithStatusStrings(ctx, p.Matricula, []string{"pending", "approved"})
		if err != nil {
			return err
		}
		if inTeam {
			return ErrParticipantInActiveTeam
		}
	}

	if s.reserveRemover != nil {
		if err := s.reserveRemover.RemoveByParticipant(ctx, id); err != nil {
			return err
		}
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) CancelByMatricula(ctx context.Context, matricula string) error {
	p, err := s.GetByMatricula(ctx, matricula)
	if err != nil {
		return err
	}
	return s.Cancel(ctx, p.ID)
}

func (s *Service) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]Participant, error) {
	return s.repo.FindByIDs(ctx, ids)
}
