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

func (s *UsuarioService) publishAudit(ctx context.Context, actorID uuid.UUID, op models.AuditOperation, targetID uuid.UUID, ip string, err error, im models.ImportanceLevel) {
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
		IPOrigen:       &ip,
		Detalles:       detalles,
		FechaOperacion: time.Now(),
		Gravedad:       im,
	})
}

func (s *UsuarioService) Create(ctx context.Context, actorID uuid.UUID, ip string, req models.CreateUsuarioAdminRequest) (models.Usuario, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		return models.Usuario{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.repo.Create(ctx, req, string(hashedPassword))
	s.publishAudit(ctx, actorID, models.AuditCreate, user.ID, ip, err, models.Informative)
	return user, err
}

func (s *UsuarioService) Update(ctx context.Context, targetID, actorID uuid.UUID, ip string, req models.UpdateUserRequest) error {
	err := s.repo.Update(ctx, targetID, req)

	s.publishAudit(ctx, actorID, models.AuditUpdate, targetID, ip, err, models.High)
	return err
}

func (s *UsuarioService) Delete(ctx context.Context, targetID, actorID uuid.UUID, ip string) error {
	if targetID == actorID {
		err := ErrCannotDeleteSelf
		s.publishAudit(ctx, actorID, models.AuditDelete, targetID, ip, err, models.Critical)
		return err
	}
	err := s.repo.Deactivate(ctx, targetID)
	s.publishAudit(ctx, actorID, models.AuditDelete, targetID, ip, err, models.High)
	return err
}

func (s *UsuarioService) AssignRole(ctx context.Context, targetID, actorID uuid.UUID, ip string, role string) error {
	err := s.repo.AssignRole(ctx, targetID, role)
	s.publishAudit(ctx, actorID, models.AuditChangePermission, targetID, ip, err, models.High)
	return err
}

func (s *UsuarioService) List(ctx context.Context, filters repositories.UsuarioFilters) ([]models.Usuario, int, error) {
	return s.repo.List(ctx, filters)
}

func (s *UsuarioService) UsuariosActivos(ctx context.Context) (int, error) {
	return s.repo.UsuariosActivos(ctx)
}
