package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
)

type FormulaHandler struct {
	pool *pgxpool.Pool
}

func NewFormulaHandler(pool *pgxpool.Pool) *FormulaHandler {
	return &FormulaHandler{pool: pool}
}

// resolveMedico devuelve el medico_id del usuario autenticado, o error HTTP ya escrito.
func (h *FormulaHandler) resolveMedico(c *gin.Context) (uuid.UUID, bool) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return uuid.Nil, false
	}
	var medicoID uuid.UUID
	err = h.pool.QueryRow(context.Background(),
		`SELECT id FROM medico WHERE usuario_id = $1`, userID).Scan(&medicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede gestionar fórmulas"})
			return uuid.Nil, false
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return uuid.Nil, false
	}
	return medicoID, true
}

// Create maneja POST /api/v1/formulas (HU-06).
// El médico registra una fórmula ligada a una consulta del paciente.
func (h *FormulaHandler) Create(c *gin.Context) {
	medicoID, ok := h.resolveMedico(c)
	if !ok {
		return
	}

	var req models.CreateFormulaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pacienteID, _ := uuid.Parse(req.PacienteID)
	consultaID, _ := uuid.Parse(req.ConsultaID)

	var fechaVencimiento *time.Time
	if req.FechaVencimiento != nil && *req.FechaVencimiento != "" {
		parsed, err := time.Parse("2006-01-02", *req.FechaVencimiento)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fecha_vencimiento debe tener formato YYYY-MM-DD"})
			return
		}
		fechaVencimiento = &parsed
	}

	ctx := context.Background()

	// El paciente debe estar bajo el cuidado del médico y tener historia clínica.
	var historiaClinicaID uuid.UUID
	err := h.pool.QueryRow(ctx,
		`SELECT id FROM historia_clinica WHERE paciente_id = $1 AND medico_tratante_id = $2`,
		pacienteID, medicoID,
	).Scan(&historiaClinicaID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "el paciente no está bajo su cuidado"})
			return
		}
		log.Printf("lookup historia error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify paciente"})
		return
	}

	// La consulta indicada debe pertenecer a ese paciente.
	var consultaOK bool
	if err := h.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM consulta WHERE id = $1 AND paciente_id = $2)`,
		consultaID, pacienteID,
	).Scan(&consultaOK); err != nil {
		log.Printf("check consulta error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify consulta"})
		return
	}
	if !consultaOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la consulta no pertenece al paciente"})
		return
	}

	medicamentosJSON, err := json.Marshal(req.Medicamentos)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode medicamentos"})
		return
	}

	var formulaID uuid.UUID
	var fechaPrescripcion time.Time
	err = h.pool.QueryRow(ctx,
		`INSERT INTO formula_medica
		   (historia_clinica_id, paciente_id, medico_id, consulta_id, medicamentos,
		    indicaciones, contraindicaciones, fecha_vencimiento, estado_formula)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'vigente')
		 RETURNING id, fecha_prescripcion`,
		historiaClinicaID, pacienteID, medicoID, consultaID, medicamentosJSON,
		req.Indicaciones, req.Contraindicaciones, fechaVencimiento,
	).Scan(&formulaID, &fechaPrescripcion)
	if err != nil {
		log.Printf("create formula error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create fórmula"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                 formulaID,
		"paciente_id":        pacienteID,
		"consulta_id":        consultaID,
		"fecha_prescripcion": fechaPrescripcion,
	})
}

// ListByPaciente maneja GET /api/v1/pacientes/:id/formulas.
// Devuelve las fórmulas del paciente, de la más reciente a la más antigua.
func (h *FormulaHandler) ListByPaciente(c *gin.Context) {
	pacienteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	rows, err := h.pool.Query(context.Background(),
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
		log.Printf("list formulas error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fórmulas"})
		return
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
			log.Printf("scan formula error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read fórmulas"})
			return
		}
		f.MedicoNombre = nombre + " " + apellidos
		formulas = append(formulas, f)
	}

	c.JSON(http.StatusOK, gin.H{"formulas": formulas, "total": len(formulas)})
}

// Anular maneja POST /api/v1/formulas/:id/anular.
// Marca la fórmula como no vigente. Solo aplica a fórmulas de pacientes del médico.
func (h *FormulaHandler) Anular(c *gin.Context) {
	medicoID, ok := h.resolveMedico(c)
	if !ok {
		return
	}
	formulaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fórmula id"})
		return
	}

	tag, err := h.pool.Exec(context.Background(),
		`UPDATE formula_medica f
		    SET estado_formula = 'anulada'
		  FROM historia_clinica hc
		  WHERE f.id = $1
		    AND hc.paciente_id = f.paciente_id
		    AND hc.medico_tratante_id = $2`,
		formulaID, medicoID,
	)
	if err != nil {
		log.Printf("anular formula error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to anular fórmula"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "la fórmula no corresponde a uno de sus pacientes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "anulada"})
}
