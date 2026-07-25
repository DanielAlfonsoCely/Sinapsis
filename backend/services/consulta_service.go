package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sinapsis-backend/audit"
	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
)

type ConsultaService struct {
	repo      *repositories.ConsultaRepository
	publisher *audit.Publisher
}

func NewConsultaService(repo *repositories.ConsultaRepository, publisher *audit.Publisher) *ConsultaService {
	return &ConsultaService{repo: repo, publisher: publisher}
}

// publishAudit sigue el mismo patrón que UsuarioService.publishAudit: si err
// no es nil, su mensaje queda en Detalles, pero la entrada se publica siempre
// (éxito o fallo).
func (s *ConsultaService) publishAudit(ctx context.Context, actorID uuid.UUID, op models.AuditOperation, tabla string, registroID *uuid.UUID, err error) {
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

func (s *ConsultaService) ResolveMedicoID(ctx context.Context, usuarioID uuid.UUID) (uuid.UUID, error) {
	return s.repo.ResolveMedicoID(ctx, usuarioID)
}

// Create ejecuta la transacción de creación de consulta (+ fórmula anidada si
// aplica) y audita cada tabla afectada por separado: la consulta siempre, y la
// fórmula médica solo si el médico recetó medicamentos en la misma consulta.
func (s *ConsultaService) Create(ctx context.Context, actorID, medicoID uuid.UUID, req models.CreateConsultaRequest, pacienteID uuid.UUID, proximaCita *time.Time) (repositories.CreateResult, error) {
	res, err := s.repo.Create(ctx, medicoID, req, pacienteID, proximaCita)

	var consultaRegistroID *uuid.UUID
	if res.ConsultaID != uuid.Nil {
		id := res.ConsultaID
		consultaRegistroID = &id
	}
	s.publishAudit(ctx, actorID, models.AuditCreate, "consulta", consultaRegistroID, err)

	// Solo se audita la fórmula si efectivamente se llegó a crear (INSERT
	// exitoso dentro de la misma transacción).
	if err == nil && res.FormulaID != nil {
		s.publishAudit(ctx, actorID, models.AuditCreate, "formula_medica", res.FormulaID, nil)
	}

	return res, err
}

func (s *ConsultaService) UpdatePreDiagnostico(ctx context.Context, actorID, consultaID, medicoID uuid.UUID, preDiagnostico string) error {
	err := s.repo.UpdatePreDiagnostico(ctx, consultaID, medicoID, preDiagnostico)
	s.publishAudit(ctx, actorID, models.AuditUpdate, "consulta", &consultaID, err)
	return err
}

// ListByPaciente audita la lectura como 'consultar' -- es una lectura sensible
// (historial clínico completo de un paciente), a diferencia de listados
// administrativos genéricos que no se auditan hoy.
func (s *ConsultaService) ListByPaciente(ctx context.Context, actorID, pacienteID uuid.UUID) ([]models.ConsultaListItem, error) {
	items, err := s.repo.ListByPaciente(ctx, pacienteID)
	s.publishAudit(ctx, actorID, models.AuditConsult, "consulta", &pacienteID, err)
	return items, err
}
