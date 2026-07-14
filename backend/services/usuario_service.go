package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"sinapsis-backend/audit"
	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
)

var ErrCannotDeleteSelf = errors.New("cannot delete self")

type UsuarioService struct {
	repo      *repositories.UsuarioRepository
	publisher *audit.Publisher
}

func NewUsuarioService(repo *repositories.UsuarioRepository, publisher *audit.Publisher) *UsuarioService {
	return &UsuarioService{repo: repo, publisher: publisher}
}

func (s *UsuarioService) publishAudit(ctx context.Context, actorID uuid.UUID, op models.AuditOperation, targetID uuid.UUID, err error) {
	var detalles *string
	if err != nil {
		msg := err.Error()
		detalles = &msg
	}
	s.publisher.Publish(ctx, models.AuditLogEntry{
		ID:             uuid.New(),
		UsuarioID:      actorID,
		TipoOperacion:  op,
		TablaAfectada:  "usuario",
		RegistroID:     &targetID,
		Detalles:       detalles,
		FechaOperacion: time.Now(),
	})
}

func (s *UsuarioService) Create(ctx context.Context, actorID uuid.UUID, req models.RegisterRequest) (models.Usuario, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		return models.Usuario{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.repo.Create(ctx, req, string(hashedPassword))
	s.publishAudit(ctx, actorID, models.AuditCreate, user.ID, err)
	return user, err
}

func (s *UsuarioService) Update(ctx context.Context, targetID, actorID uuid.UUID, req models.UpdateUserRequest) error {
	err := s.repo.Update(ctx, targetID, req)
	s.publishAudit(ctx, actorID, models.AuditUpdate, targetID, err)
	return err
}

func (s *UsuarioService) Delete(ctx context.Context, targetID, actorID uuid.UUID) error {
	if targetID == actorID {
		err := ErrCannotDeleteSelf
		s.publishAudit(ctx, actorID, models.AuditDelete, targetID, err)
		return err
	}
	err := s.repo.Deactivate(ctx, targetID)
	s.publishAudit(ctx, actorID, models.AuditDelete, targetID, err)
	return err
}

func (s *UsuarioService) AssignRole(ctx context.Context, targetID, actorID uuid.UUID, role string) error {
	err := s.repo.AssignRole(ctx, targetID, role)
	s.publishAudit(ctx, actorID, models.AuditChangePermission, targetID, err)
	return err
}

func (s *UsuarioService) List(ctx context.Context, filters repositories.UsuarioFilters) ([]models.Usuario, int, error) {
	return s.repo.List(ctx, filters)
}
