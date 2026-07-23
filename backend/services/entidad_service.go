package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sinapsis-backend/audit"
	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
)

type EntidadService struct {
	repo      *repositories.EntidadRepository
	publisher *audit.Publisher
}

func NewEntidadService(repo *repositories.EntidadRepository, publisher *audit.Publisher) *EntidadService {
	return &EntidadService{repo: repo, publisher: publisher}
}

func (s *EntidadService) publishAudit(ctx context.Context, actorID uuid.UUID, ip string, op models.AuditOperation, tabla string, registroID *uuid.UUID, err error, im models.ImportanceLevel) {
	var detalles *string
	if err != nil {
		msg := err.Error()
		detalles = &msg
	}
	s.publisher.Publish(ctx, models.AuditLogEntry{
		ID:             uuid.New(),
		UsuarioID:      actorID,
		TipoOperacion:  op,
		TablaAfectada:  tabla,
		RegistroID:     registroID,
		Detalles:       detalles,
		IPOrigen:       &ip,
		FechaOperacion: time.Now(),
		Gravedad:       im,
	})
}

// Create registra una nueva entidad. Solo accesible por admin (ver handler/routes).
func (s *EntidadService) Create(ctx context.Context, actorID uuid.UUID, ip string, req models.CreateEntidadRequest) (models.Entidad, error) {
	entidad, err := s.repo.Create(ctx, req)

	var registroID *uuid.UUID
	if entidad.ID != uuid.Nil {
		id := entidad.ID
		registroID = &id
	}
	s.publishAudit(ctx, actorID, ip, models.AuditCreate, "entidad", registroID, err, models.Informative)
	return entidad, err
}

// List devuelve el listado público de entidades (sin auditoría: no expone
// datos sensibles distintos a los ya públicos en el registro de entidades).
func (s *EntidadService) List(ctx context.Context) ([]models.Entidad, error) {
	return s.repo.List(ctx)
}

// ListAdmin audita como 'consultar': es un listado administrativo.
func (s *EntidadService) ListAdmin(ctx context.Context, actorID uuid.UUID, ip string, q string) ([]models.AdminEntidadListItem, error) {
	items, err := s.repo.ListAdmin(ctx, q)
	s.publishAudit(ctx, actorID, ip, models.AuditConsult, "entidad", nil, err, models.Informative)
	return items, err
}

// GetByIDAdmin audita como 'consultar': expone usuarios asociados e
// historial de historias clínicas de la entidad, es una lectura sensible.
func (s *EntidadService) GetByIDAdmin(ctx context.Context, actorID uuid.UUID, ip string, id uuid.UUID) (models.EntidadDetalle, error) {
	detalle, err := s.repo.GetByIDAdmin(ctx, id)
	s.publishAudit(ctx, actorID, ip, models.AuditConsult, "entidad", &id, err, models.Informative)
	return detalle, err
}

// Stats devuelve métricas agregadas de la plataforma para el dashboard admin.
func (s *EntidadService) Stats(ctx context.Context, actorID uuid.UUID, ip string) (repositories.PlatformStats, error) {
	stats, err := s.repo.Stats(ctx)
	s.publishAudit(ctx, actorID, ip, models.AuditConsult, "entidad", nil, err, models.Informative)
	return stats, err
}

func (s *EntidadService) UpdateAdmin(ctx context.Context, actorID uuid.UUID, ip string, id uuid.UUID, req models.UpdateEntidadRequest) (models.Entidad, error) {
	entidad, err := s.repo.UpdateAdmin(ctx, id, req)

	var registroID *uuid.UUID
	if entidad.ID != uuid.Nil {
		registroID = &entidad.ID
	}
	s.publishAudit(ctx, actorID, ip, models.AuditUpdate, "entidad", registroID, err, models.Informative)
	return entidad, err
}
