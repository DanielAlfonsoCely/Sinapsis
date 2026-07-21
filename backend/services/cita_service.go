package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sinapsis-backend/audit"
	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
)

type CitaService struct {
	repo      *repositories.CitaRepository
	publisher *audit.Publisher
}

func NewCitaService(repo *repositories.CitaRepository, publisher *audit.Publisher) *CitaService {
	return &CitaService{repo: repo, publisher: publisher}
}

func (s *CitaService) publishAudit(ctx context.Context, actorID uuid.UUID, op models.AuditOperation, tabla string, registroID *uuid.UUID, err error, im models.ImportanceLevel) {
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
		Gravedad:       im,
	})
}

func (s *CitaService) ResolveMedicoID(ctx context.Context, usuarioID uuid.UUID) (uuid.UUID, error) {
	return s.repo.ResolveMedicoID(ctx, usuarioID)
}

// Create agenda la cita y audita la escritura. Nota de alcance: CitasHoy,
// CitasSemana y Disponibilidad (ver abajo) NO se auditan -- son lecturas del
// médico sobre su propia agenda, no "consultar el expediente de un paciente
// específico" como sí ocurre con Consulta/Formula.ListByPaciente.
func (s *CitaService) Create(ctx context.Context, actorID, usuarioID, medicoID uuid.UUID, fechaHora time.Time, motivo *string) (repositories.CreateCitaResult, error) {
	res, err := s.repo.Create(ctx, usuarioID, medicoID, fechaHora, motivo)

	var registroID *uuid.UUID
	if res.CitaID != uuid.Nil {
		id := res.CitaID
		registroID = &id
	}
	s.publishAudit(ctx, actorID, models.AuditCreate, "cita", registroID, err, models.Informative)
	return res, err
}

func (s *CitaService) CitasHoy(ctx context.Context, medicoID uuid.UUID) ([]repositories.CitaHoyItem, error) {
	return s.repo.CitasHoy(ctx, medicoID)
}

func (s *CitaService) CitasSemana(ctx context.Context, medicoID uuid.UUID, lunes, domingo time.Time) ([]repositories.CitaSemanaItem, error) {
	return s.repo.CitasSemana(ctx, medicoID, lunes, domingo)
}

func (s *CitaService) ExisteMedicoActivo(ctx context.Context, medicoID uuid.UUID) (bool, error) {
	return s.repo.ExisteMedicoActivo(ctx, medicoID)
}

func (s *CitaService) HorariosOcupados(ctx context.Context, medicoID uuid.UUID, fecha, diaSiguiente time.Time) (map[string]bool, error) {
	return s.repo.HorariosOcupados(ctx, medicoID, fecha, diaSiguiente)
}
