package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sinapsis-backend/audit"
	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
)

type FormulaService struct {
	repo      *repositories.FormulaRepository
	publisher *audit.Publisher
}

func NewFormulaService(repo *repositories.FormulaRepository, publisher *audit.Publisher) *FormulaService {
	return &FormulaService{repo: repo, publisher: publisher}
}

func (s *FormulaService) publishAudit(ctx context.Context, actorID uuid.UUID, op models.AuditOperation, tabla string, registroID *uuid.UUID, err error) {
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
		FechaOperacion: time.Now(),
	})
}

func (s *FormulaService) ResolveMedicoID(ctx context.Context, usuarioID uuid.UUID) (uuid.UUID, error) {
	return s.repo.ResolveMedicoID(ctx, usuarioID)
}

func (s *FormulaService) Create(ctx context.Context, actorID, medicoID, pacienteID, consultaID uuid.UUID, req models.CreateFormulaRequest, fechaVencimiento *time.Time) (repositories.CreateFormulaResult, error) {
	res, err := s.repo.Create(ctx, medicoID, pacienteID, consultaID, req, fechaVencimiento)

	var registroID *uuid.UUID
	if res.FormulaID != uuid.Nil {
		id := res.FormulaID
		registroID = &id
	}
	s.publishAudit(ctx, actorID, models.AuditCreate, "formula_medica", registroID, err)
	return res, err
}

// ListByPaciente audita la lectura como 'consultar' -- ver las fórmulas de un
// paciente es una lectura clínica sensible.
func (s *FormulaService) ListByPaciente(ctx context.Context, actorID, pacienteID uuid.UUID) ([]models.FormulaListItem, error) {
	items, err := s.repo.ListByPaciente(ctx, pacienteID)
	s.publishAudit(ctx, actorID, models.AuditConsult, "formula_medica", &pacienteID, err)
	return items, err
}

// Anular usa 'eliminar' como tipo_operacion: el enum tipo_operacion_enum no
// tiene un valor específico para "anular" (baja lógica), y 'eliminar' es el
// más cercano semánticamente disponible hoy en el schema.
func (s *FormulaService) Anular(ctx context.Context, actorID, formulaID, medicoID uuid.UUID) error {
	err := s.repo.Anular(ctx, formulaID, medicoID)
	s.publishAudit(ctx, actorID, models.AuditDelete, "formula_medica", &formulaID, err)
	return err
}
