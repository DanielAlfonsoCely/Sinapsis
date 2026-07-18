package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
)

var (
	ErrSoloMedico           = errors.New("solo un médico puede realizar esta operación")
	ErrSinCitaHoy           = errors.New("el paciente no tiene una cita activa para hoy")
	ErrHistoriaNoExiste     = errors.New("el paciente no tiene historia clínica")
	ErrConsultaNoAutorizada = errors.New("consulta no encontrada o no autorizada")
)

type ConsultaRepository struct {
	pool *pgxpool.Pool
}

func NewConsultaRepository(pool *pgxpool.Pool) *ConsultaRepository {
	return &ConsultaRepository{pool: pool}
}

// ResolveMedicoID devuelve el medico_id asociado a un usuario autenticado.
// Reutilizado por Create y UpdatePreDiagnostico, igual que en el handler original.
func (r *ConsultaRepository) ResolveMedicoID(ctx context.Context, usuarioID uuid.UUID) (uuid.UUID, error) {
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

// CreateResult agrupa lo que Create necesita devolver para que el service
// pueda auditar tanto la consulta como (si aplica) la fórmula generada.
type CreateResult struct {
	ConsultaID            uuid.UUID
	HistoriaClinicaID     uuid.UUID
	PacienteID            uuid.UUID
	FechaConsulta         time.Time
	FormulaID             *uuid.UUID
	CitaMarcadaCompletada bool
}

// Create replica exactamente la transacción que antes vivía en
// ConsultaHandler.Create: valida cita activa de hoy, obtiene la historia
// clínica, inserta la consulta, inserta la fórmula si vino con medicamentos,
// y marca la cita de hoy como completada. Ninguna regla de negocio cambió.
func (r *ConsultaRepository) Create(ctx context.Context, medicoID uuid.UUID, req models.CreateConsultaRequest, pacienteID uuid.UUID, proximaCita *time.Time) (CreateResult, error) {
	var res CreateResult
	res.PacienteID = pacienteID

	var tieneCitaHoy bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM cita
		   WHERE paciente_id = $1 AND medico_id = $2
		     AND estado = 'programada' AND fecha_hora::date = CURRENT_DATE)`,
		pacienteID, medicoID,
	).Scan(&tieneCitaHoy); err != nil {
		return res, err
	}
	if !tieneCitaHoy {
		return res, ErrSinCitaHoy
	}

	var historiaClinicaID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM historia_clinica WHERE paciente_id = $1`, pacienteID,
	).Scan(&historiaClinicaID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return res, ErrHistoriaNoExiste
		}
		return res, err
	}
	res.HistoriaClinicaID = historiaClinicaID

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)

	var consultaID uuid.UUID
	var fechaConsulta time.Time
	err = tx.QueryRow(ctx,
		`INSERT INTO consulta (
			historia_clinica_id, paciente_id, medico_id,
			tipo_consulta, motivo_consulta, anamnesis, revision_sistemas,
			examen_fisico, hallazgos_clinicos,
			presion_arterial, frecuencia_cardiaca, frecuencia_respiratoria,
			temperatura, saturacion_oxigeno, peso_kg, talla_cm,
			diagnostico_principal, diagnostico_cie10, plan_manejo,
			procedimientos_indicados, observaciones_medico, pre_diagnostico,
			proxima_cita, fecha_consulta, estado_consulta
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9,
			$10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19,
			$20, $21, $22,
			$23, NOW(), 'completada'
		) RETURNING id, fecha_consulta`,
		historiaClinicaID, pacienteID, medicoID,
		req.TipoConsulta, req.MotivoConsulta, req.Anamnesis, req.RevisionSistemas,
		req.ExamenFisico, req.HallazgosClinicos,
		req.PresionArterial, req.FrecuenciaCardiaca, req.FrecuenciaRespiratoria,
		req.Temperatura, req.SaturacionOxigeno, req.PesoKg, req.TallaCm,
		req.DiagnosticoPrincipal, req.DiagnosticoCIE10, req.PlanManejo,
		req.ProcedimientosIndicados, req.ObservacionesMedico, req.PreDiagnostico,
		proximaCita,
	).Scan(&consultaID, &fechaConsulta)
	if err != nil {
		return res, err
	}
	res.ConsultaID = consultaID
	res.FechaConsulta = fechaConsulta

	if len(req.Medicamentos) > 0 {
		medicamentosJSON, err := json.Marshal(req.Medicamentos)
		if err != nil {
			return res, err
		}
		var fid uuid.UUID
		err = tx.QueryRow(ctx,
			`INSERT INTO formula_medica
			   (historia_clinica_id, paciente_id, medico_id, consulta_id, medicamentos,
			    indicaciones, estado_formula)
			 VALUES ($1, $2, $3, $4, $5, $6, 'vigente')
			 RETURNING id`,
			historiaClinicaID, pacienteID, medicoID, consultaID, medicamentosJSON,
			req.FormulaIndicaciones,
		).Scan(&fid)
		if err != nil {
			return res, err
		}
		res.FormulaID = &fid
	}

	tag, err := tx.Exec(ctx,
		`UPDATE cita SET estado = 'completada'
		  WHERE paciente_id = $1 AND medico_id = $2
		    AND estado = 'programada' AND fecha_hora::date = CURRENT_DATE`,
		pacienteID, medicoID,
	)
	if err != nil {
		return res, err
	}
	res.CitaMarcadaCompletada = tag.RowsAffected() > 0

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	return res, nil
}

// UpdatePreDiagnostico replica el UPDATE original: solo el médico que atendió
// la consulta puede registrar su pre-diagnóstico.
func (r *ConsultaRepository) UpdatePreDiagnostico(ctx context.Context, consultaID, medicoID uuid.UUID, preDiagnostico string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE consulta SET pre_diagnostico = $1
		  WHERE id = $2 AND medico_id = $3`,
		preDiagnostico, consultaID, medicoID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConsultaNoAutorizada
	}
	return nil
}

// ListByPaciente replica exactamente el SELECT original (join con médico/usuario,
// anexos y sugerencias IA agregadas como JSON).
func (r *ConsultaRepository) ListByPaciente(ctx context.Context, pacienteID uuid.UUID) ([]models.ConsultaListItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.tipo_consulta, c.motivo_consulta, c.anamnesis, c.revision_sistemas,
		        c.examen_fisico, c.hallazgos_clinicos,
		        c.presion_arterial, c.frecuencia_cardiaca, c.frecuencia_respiratoria,
		        c.temperatura, c.saturacion_oxigeno, c.peso_kg, c.talla_cm,
		        c.diagnostico_principal, c.diagnostico_cie10, c.plan_manejo,
		        c.procedimientos_indicados, c.observaciones_medico, c.pre_diagnostico,
		        c.proxima_cita, c.fecha_consulta, c.estado_consulta,
		        u.nombre_usuario, u.apellidos, m.especialidad,
		        COALESCE((
		          SELECT json_agg(json_build_object('id', e.id, 'nombre', e.descripcion, 'tipo', e.tipo_examen)
		                          ORDER BY e.fecha_carga)
		          FROM examinagen e WHERE e.consulta_id = c.id
		        ), '[]') AS anexos,
		        COALESCE((
		          SELECT json_agg(json_build_object(
		                   'id', s.id,
		                   'examinagen_id', s.examinagen_id,
		                   'modelo_ia_utilizado', s.modelo_ia_utilizado,
		                   'estado_procesamiento', s.estado_procesamiento,
		                   'diagnostico_sugerido', s.diagnostico_sugerido,
		                   'descripcion_hallazgo', s.descripcion_hallazgo,
		                   'estado_revision', s.estado_revision
		                 ) ORDER BY s.fecha_analisis DESC)
		          FROM sugerencia_ia s
		          JOIN examinagen ex ON ex.id = s.examinagen_id
		          WHERE ex.consulta_id = c.id
		        ), '[]') AS sugerencias_ia
		 FROM consulta c
		 JOIN medico m ON m.id = c.medico_id
		 JOIN usuario u ON u.id = m.usuario_id
		 WHERE c.paciente_id = $1
		 ORDER BY c.fecha_consulta DESC`,
		pacienteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	consultas := make([]models.ConsultaListItem, 0)
	for rows.Next() {
		var item models.ConsultaListItem
		var nombre, apellidos string
		var anexosJSON, sugerenciasJSON []byte
		if err := rows.Scan(
			&item.ID, &item.TipoConsulta, &item.MotivoConsulta, &item.Anamnesis, &item.RevisionSistemas,
			&item.ExamenFisico, &item.HallazgosClinicos,
			&item.PresionArterial, &item.FrecuenciaCardiaca, &item.FrecuenciaRespiratoria,
			&item.Temperatura, &item.SaturacionOxigeno, &item.PesoKg, &item.TallaCm,
			&item.DiagnosticoPrincipal, &item.DiagnosticoCIE10, &item.PlanManejo,
			&item.ProcedimientosIndicados, &item.ObservacionesMedico, &item.PreDiagnostico,
			&item.ProximaCita, &item.FechaConsulta, &item.EstadoConsulta,
			&nombre, &apellidos, &item.MedicoEspecialidad, &anexosJSON, &sugerenciasJSON,
		); err != nil {
			return nil, err
		}
		item.MedicoNombre = nombre + " " + apellidos
		if err := json.Unmarshal(anexosJSON, &item.Anexos); err != nil {
			item.Anexos = []models.AnexoItem{}
		}
		if err := json.Unmarshal(sugerenciasJSON, &item.SugerenciasIA); err != nil {
			item.SugerenciasIA = []models.SugerenciaIAItem{}
		}
		consultas = append(consultas, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return consultas, nil
}
