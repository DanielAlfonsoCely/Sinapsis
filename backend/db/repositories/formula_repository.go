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
	ErrPacienteNoACargo    = errors.New("el paciente no está bajo su cuidado")
	ErrConsultaNoPertenece = errors.New("la consulta no pertenece al paciente")
	ErrFormulaNoAutorizada = errors.New("la fórmula no corresponde a uno de sus pacientes")
)

type FormulaRepository struct {
	pool *pgxpool.Pool
}

func NewFormulaRepository(pool *pgxpool.Pool) *FormulaRepository {
	return &FormulaRepository{pool: pool}
}

func (r *FormulaRepository) ResolveMedicoID(ctx context.Context, usuarioID uuid.UUID) (uuid.UUID, error) {
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

type CreateFormulaResult struct {
	FormulaID         uuid.UUID
	FechaPrescripcion time.Time
}

// Create replica exactamente las validaciones e INSERT que antes vivían en
// FormulaHandler.Create: el paciente debe estar bajo el cuidado del médico y
// la consulta indicada debe pertenecerle.
func (r *FormulaRepository) Create(ctx context.Context, medicoID, pacienteID, consultaID uuid.UUID, req models.CreateFormulaRequest, fechaVencimiento *time.Time) (CreateFormulaResult, error) {
	var res CreateFormulaResult

	var historiaClinicaID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM historia_clinica WHERE paciente_id = $1 AND medico_tratante_id = $2`,
		pacienteID, medicoID,
	).Scan(&historiaClinicaID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return res, ErrPacienteNoACargo
		}
		return res, err
	}

	var consultaOK bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM consulta WHERE id = $1 AND paciente_id = $2)`,
		consultaID, pacienteID,
	).Scan(&consultaOK); err != nil {
		return res, err
	}
	if !consultaOK {
		return res, ErrConsultaNoPertenece
	}

	medicamentosJSON, err := json.Marshal(req.Medicamentos)
	if err != nil {
		return res, err
	}

	var formulaID uuid.UUID
	var fechaPrescripcion time.Time
	err = r.pool.QueryRow(ctx,
		`INSERT INTO formula_medica
		   (historia_clinica_id, paciente_id, medico_id, consulta_id, medicamentos,
		    indicaciones, contraindicaciones, fecha_vencimiento, estado_formula)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'vigente')
		 RETURNING id, fecha_prescripcion`,
		historiaClinicaID, pacienteID, medicoID, consultaID, medicamentosJSON,
		req.Indicaciones, req.Contraindicaciones, fechaVencimiento,
	).Scan(&formulaID, &fechaPrescripcion)
	if err != nil {
		return res, err
	}
	res.FormulaID = formulaID
	res.FechaPrescripcion = fechaPrescripcion
	return res, nil
}

func (r *FormulaRepository) ListByPaciente(ctx context.Context, pacienteID uuid.UUID) ([]models.FormulaListItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT f.id, f.consulta_id, f.medicamentos, f.indicaciones, f.contraindicaciones,
		        f.fecha_prescripcion, f.fecha_vencimiento, f.estado_formula,
		        u.nombre_usuario, u.apellidos
		 FROM formula_medica f
		 JOIN medico m ON m.id = f.medico_id
		 JOIN usuario u ON u.id = m.usuario_id
		 WHERE f.paciente_id = $1
		 ORDER BY f.fecha_prescripcion DESC`,
		pacienteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	formulas := make([]models.FormulaListItem, 0)
	for rows.Next() {
		var f models.FormulaListItem
		var nombre, apellidos string
		if err := rows.Scan(
			&f.ID, &f.ConsultaID, &f.Medicamentos, &f.Indicaciones, &f.Contraindicaciones,
			&f.FechaPrescripcion, &f.FechaVencimiento, &f.EstadoFormula,
			&nombre, &apellidos,
		); err != nil {
			return nil, err
		}
		f.MedicoNombre = nombre + " " + apellidos
		formulas = append(formulas, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return formulas, nil
}

// Anular replica el UPDATE original: marca la fórmula anulada solo si
// pertenece a un paciente del médico autenticado.
func (r *FormulaRepository) Anular(ctx context.Context, formulaID, medicoID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE formula_medica f
		    SET estado_formula = 'anulada'
		  FROM historia_clinica hc
		  WHERE f.id = $1
		    AND hc.paciente_id = f.paciente_id
		    AND hc.medico_tratante_id = $2`,
		formulaID, medicoID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFormulaNoAutorizada
	}
	return nil
}
