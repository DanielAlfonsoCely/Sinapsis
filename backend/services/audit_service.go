package services

import (
	"context"

	"github.com/google/uuid"

	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
)

type AuditService struct {
	repo *repositories.AuditRepository
}

func NewAuditService(repo *repositories.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Record(
	ctx context.Context,
	usuarioID uuid.UUID,
	operacion models.AuditOperation,
	tabla string,
	registroID *uuid.UUID,
	ipOrigen *string,
	detalles *string,
) error {
	return s.repo.Record(ctx, usuarioID, operacion, tabla, registroID, ipOrigen, detalles)
}

func (s *AuditService) ListRecent(ctx context.Context, limit, offset int) ([]models.AuditLogEntry, int, error) {
	return s.repo.ListRecent(ctx, limit, offset)
}
func (s *AuditService) LookCritical(ctx context.Context, limit, offset int) ([]models.AuditLogEntry, int, error) {
	return s.repo.LookCritical(ctx, limit, offset)
}
