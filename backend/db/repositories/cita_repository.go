package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMedicoNoExiste      = errors.New("el médico no existe")
	ErrSinAutorizacion     = errors.New("no tienes autorización para agendar con esta especialidad")
	ErrCitaActivaExistente = errors.New("ya tienes una cita activa con este médico")
	ErrHorarioOcupado      = errors.New("ese horario ya fue tomado, elige otro")
	ErrSoloPaciente        = errors.New("solo un paciente puede agendar sus citas")
)

type CitaRepository struct {
	pool *pgxpool.Pool
}

func NewCitaRepository(pool *pgxpool.Pool) *CitaRepository {
	return &CitaRepository{pool: pool}
}

type PacienteResumen struct {
	ID                string `json:"id"`
	NombrePaciente    string `json:"nombre_paciente"`
	ApellidosPaciente string `json:"apellidos_paciente"`
	NumeroDocumento   string `json:"numero_documento"`
}

type CitaHoyItem struct {
	ID        string          `json:"id"`
	FechaHora time.Time       `json:"fecha_hora"`
	Estado    string          `json:"estado"`
	Paciente  PacienteResumen `json:"paciente"`
}

type CitaSemanaItem struct {
	ID        string          `json:"id"`
	FechaHora time.Time       `json:"fecha_hora"`
	Estado    string          `json:"estado"`
	Motivo    *string         `json:"motivo"`
	Paciente  PacienteResumen `json:"paciente"`
}

func (r *CitaRepository) ResolveMedicoID(ctx context.Context, usuarioID uuid.UUID) (uuid.UUID, error) {
	var medicoID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM medico WHERE usuario_id = $1`, usuarioID,
	).Scan(&medicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrSoloMedico
		}
		return uuid.Nil, err
	}
	return medicoID, nil
}

func (r *CitaRepository) CitasHoy(ctx context.Context, medicoID uuid.UUID) ([]CitaHoyItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ci.id, ci.fecha_hora, ci.estado,
		        p.id, p.nombre_paciente, p.apellidos_paciente, p.numero_documento
		 FROM cita ci
		 JOIN paciente p ON p.id = ci.paciente_id
		 WHERE ci.medico_id = $1
		   AND ci.fecha_hora::date = CURRENT_DATE
		 ORDER BY ci.fecha_hora`,
		medicoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	citas := make([]CitaHoyItem, 0)
	for rows.Next() {
		var ci CitaHoyItem
		if err := rows.Scan(
			&ci.ID, &ci.FechaHora, &ci.Estado,
			&ci.Paciente.ID, &ci.Paciente.NombrePaciente, &ci.Paciente.ApellidosPaciente, &ci.Paciente.NumeroDocumento,
		); err != nil {
			return nil, err
		}
		citas = append(citas, ci)
	}
	return citas, rows.Err()
}

func (r *CitaRepository) CitasSemana(ctx context.Context, medicoID uuid.UUID, lunes, domingo time.Time) ([]CitaSemanaItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ci.id, ci.fecha_hora, ci.estado, ci.motivo,
		        p.id, p.nombre_paciente, p.apellidos_paciente, p.numero_documento
		 FROM cita ci
		 JOIN paciente p ON p.id = ci.paciente_id
		 WHERE ci.medico_id = $1
		   AND ci.fecha_hora >= $2
		   AND ci.fecha_hora < $3
		 ORDER BY ci.fecha_hora`,
		medicoID, lunes, domingo,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	citas := make([]CitaSemanaItem, 0)
	for rows.Next() {
		var ci CitaSemanaItem
		if err := rows.Scan(
			&ci.ID, &ci.FechaHora, &ci.Estado, &ci.Motivo,
			&ci.Paciente.ID, &ci.Paciente.NombrePaciente, &ci.Paciente.ApellidosPaciente, &ci.Paciente.NumeroDocumento,
		); err != nil {
			return nil, err
		}
		citas = append(citas, ci)
	}
	return citas, rows.Err()
}

func (r *CitaRepository) ExisteMedicoActivo(ctx context.Context, medicoID uuid.UUID) (bool, error) {
	var existe bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM medico WHERE id = $1 AND estado = true)`, medicoID,
	).Scan(&existe)
	return existe, err
}

func (r *CitaRepository) HorariosOcupados(ctx context.Context, medicoID uuid.UUID, fecha, diaSiguiente time.Time) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT fecha_hora FROM cita
		 WHERE medico_id = $1 AND fecha_hora >= $2 AND fecha_hora < $3
		   AND estado IN ('programada','en_curso')`,
		medicoID, fecha, diaSiguiente,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ocupados := make(map[string]bool)
	for rows.Next() {
		var fh time.Time
		if err := rows.Scan(&fh); err != nil {
			return nil, err
		}
		ocupados[fh.Format("15:04")] = true
	}
	return ocupados, rows.Err()
}

// CreateResult trae lo necesario para responder al handler y para auditar.
type CreateCitaResult struct {
	CitaID       uuid.UUID
	PacienteID   uuid.UUID
	EsTratante   bool
	Especialidad string
}

// Create replica exactamente las validaciones y el INSERT que antes vivían en
// CitaHandler.Create: resuelve paciente/tratante desde el JWT, valida remisión
// si el médico destino no es el tratante, valida cita activa duplicada y
// choque de horario, e inserta la cita.
func (r *CitaRepository) Create(ctx context.Context, usuarioID, medicoID uuid.UUID, fechaHora time.Time, motivo *string) (CreateCitaResult, error) {
	var res CreateCitaResult

	var pacienteID uuid.UUID
	var tratanteID *uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT p.id, hc.medico_tratante_id
		 FROM paciente p JOIN historia_clinica hc ON hc.paciente_id = p.id
		 WHERE p.usuario_id = $1`, usuarioID,
	).Scan(&pacienteID, &tratanteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return res, ErrSoloPaciente
		}
		return res, err
	}
	res.PacienteID = pacienteID

	var especialidad string
	if err := r.pool.QueryRow(ctx,
		`SELECT especialidad FROM medico WHERE id = $1 AND estado = true`, medicoID,
	).Scan(&especialidad); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return res, ErrMedicoNoExiste
		}
		return res, err
	}
	res.Especialidad = especialidad

	esTratante := tratanteID != nil && *tratanteID == medicoID
	res.EsTratante = esTratante
	if !esTratante {
		var autorizado bool
		if err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM remision
			   WHERE paciente_id = $1 AND estado = 'autorizada' AND especialidad = $2)`,
			pacienteID, especialidad,
		).Scan(&autorizado); err != nil {
			return res, err
		}
		if !autorizado {
			return res, ErrSinAutorizacion
		}
	}

	var yaTieneCita bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM cita
		   WHERE paciente_id = $1 AND medico_id = $2 AND estado IN ('programada','en_curso'))`,
		pacienteID, medicoID,
	).Scan(&yaTieneCita); err != nil {
		return res, err
	}
	if yaTieneCita {
		return res, ErrCitaActivaExistente
	}

	var horarioOcupado bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM cita
		   WHERE medico_id = $1 AND fecha_hora = $2 AND estado IN ('programada','en_curso'))`,
		medicoID, fechaHora,
	).Scan(&horarioOcupado); err != nil {
		return res, err
	}
	if horarioOcupado {
		return res, ErrHorarioOcupado
	}

	var citaID uuid.UUID
	err = r.pool.QueryRow(ctx,
		`INSERT INTO cita (paciente_id, medico_id, fecha_hora, motivo, estado)
		 VALUES ($1, $2, $3, $4, 'programada')
		 RETURNING id`,
		pacienteID, medicoID, fechaHora, motivo,
	).Scan(&citaID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return res, ErrHorarioOcupado
		}
		return res, err
	}
	res.CitaID = citaID
	return res, nil
}
