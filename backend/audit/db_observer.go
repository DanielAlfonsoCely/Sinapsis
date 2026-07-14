package audit

import (
	"context"
	"log"

	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
)

type DBAuditObserver struct {
	repo *repositories.AuditRepository
}

func NewDBAuditObserver(repo *repositories.AuditRepository) *DBAuditObserver {
	return &DBAuditObserver{repo: repo}
}

func (o *DBAuditObserver) Notify(ctx context.Context, entry models.AuditLogEntry) {
	if err := o.repo.Insert(ctx, entry); err != nil {
		// No propagamos el error: un fallo al auditar no debe tumbar la operación original.
		log.Printf("audit insert error: %v", err)
	}
}
