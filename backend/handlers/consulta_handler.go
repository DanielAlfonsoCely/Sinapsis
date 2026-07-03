package handlers

import (
	"context"
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

type ConsultaHandler struct {
	pool *pgxpool.Pool
}

func NewConsultaHandler(pool *pgxpool.Pool) *ConsultaHandler {
	return &ConsultaHandler{pool: pool}
}

// Create maneja POST /api/v1/consultas (HU-03).
// El médico autenticado registra una nueva consulta dentro de la historia
// clínica del paciente. La historia clínica ya existe (se crea al registrar el
// paciente, HU-02), por lo que aquí solo se agrega un nuevo registro.
func (h *ConsultaHandler) Create(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	var req models.CreateConsultaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pacienteID, err := uuid.Parse(req.PacienteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paciente_id inválido"})
		return
	}

	var proximaCita *time.Time
	if req.ProximaCita != nil && *req.ProximaCita != "" {
		parsed, err := time.Parse("2006-01-02", *req.ProximaCita)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "proxima_cita debe tener formato YYYY-MM-DD"})
			return
		}
		proximaCita = &parsed
	}

	ctx := context.Background()

	// Solo un médico puede registrar consultas; su id se resuelve desde el JWT.
	var medicoID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT id FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede registrar consultas"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	// La historia clínica del paciente ya debe existir (HU-02).
	var historiaClinicaID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT id FROM historia_clinica WHERE paciente_id = $1`, pacienteID,
	).Scan(&historiaClinicaID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "el paciente no tiene historia clínica"})
			return
		}
		log.Printf("lookup historia clinica error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find historia clínica"})
		return
	}

	var consultaID uuid.UUID
	var fechaConsulta time.Time
	err = h.pool.QueryRow(ctx,
		`INSERT INTO consulta (
			historia_clinica_id, paciente_id, medico_id,
			tipo_consulta, motivo_consulta, anamnesis, revision_sistemas,
			examen_fisico, hallazgos_clinicos,
			presion_arterial, frecuencia_cardiaca, frecuencia_respiratoria,
			temperatura, saturacion_oxigeno, peso_kg, talla_cm,
			diagnostico_principal, diagnostico_cie10, plan_manejo,
			procedimientos_indicados, observaciones_medico,
			proxima_cita, fecha_consulta, estado_consulta
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9,
			$10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19,
			$20, $21,
			$22, NOW(), 'completada'
		) RETURNING id, fecha_consulta`,
		historiaClinicaID, pacienteID, medicoID,
		req.TipoConsulta, req.MotivoConsulta, req.Anamnesis, req.RevisionSistemas,
		req.ExamenFisico, req.HallazgosClinicos,
		req.PresionArterial, req.FrecuenciaCardiaca, req.FrecuenciaRespiratoria,
		req.Temperatura, req.SaturacionOxigeno, req.PesoKg, req.TallaCm,
		req.DiagnosticoPrincipal, req.DiagnosticoCIE10, req.PlanManejo,
		req.ProcedimientosIndicados, req.ObservacionesMedico,
		proximaCita,
	).Scan(&consultaID, &fechaConsulta)
	if err != nil {
		log.Printf("create consulta error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create consulta"})
		return
	}

	// La consulta atiende la cita programada de hoy: se marca como completada.
	if _, err := h.pool.Exec(ctx,
		`UPDATE cita SET estado = 'completada'
		  WHERE paciente_id = $1 AND medico_id = $2
		    AND estado = 'programada' AND fecha_hora::date = CURRENT_DATE`,
		pacienteID, medicoID,
	); err != nil {
		// No es crítico para la consulta ya registrada; solo se registra el error.
		log.Printf("marcar cita completada error: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                  consultaID,
		"historia_clinica_id": historiaClinicaID,
		"paciente_id":         pacienteID,
		"fecha_consulta":      fechaConsulta,
	})
}

// ListByPaciente maneja GET /api/v1/pacientes/:id/consultas (HU-04).
// Devuelve el historial de consultas del paciente, de la más reciente a la más
// antigua, incluyendo el médico que atendió cada consulta.
func (h *ConsultaHandler) ListByPaciente(c *gin.Context) {
	idParam := c.Param("id")
	pacienteID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	rows, err := h.pool.Query(context.Background(),
		`SELECT c.id, c.tipo_consulta, c.motivo_consulta, c.anamnesis, c.revision_sistemas,
		        c.examen_fisico, c.hallazgos_clinicos,
		        c.presion_arterial, c.frecuencia_cardiaca, c.frecuencia_respiratoria,
		        c.temperatura, c.saturacion_oxigeno, c.peso_kg, c.talla_cm,
		        c.diagnostico_principal, c.diagnostico_cie10, c.plan_manejo,
		        c.procedimientos_indicados, c.observaciones_medico,
		        c.proxima_cita, c.fecha_consulta, c.estado_consulta,
		        u.nombre_usuario, u.apellidos, m.especialidad
		 FROM consulta c
		 JOIN medico m ON m.id = c.medico_id
		 JOIN usuario u ON u.id = m.usuario_id
		 WHERE c.paciente_id = $1
		 ORDER BY c.fecha_consulta DESC`,
		pacienteID,
	)
	if err != nil {
		log.Printf("list consultas error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch consultas"})
		return
	}
	defer rows.Close()

	consultas := make([]models.ConsultaListItem, 0)
	for rows.Next() {
		var item models.ConsultaListItem
		var nombre, apellidos string
		if err := rows.Scan(
			&item.ID, &item.TipoConsulta, &item.MotivoConsulta, &item.Anamnesis, &item.RevisionSistemas,
			&item.ExamenFisico, &item.HallazgosClinicos,
			&item.PresionArterial, &item.FrecuenciaCardiaca, &item.FrecuenciaRespiratoria,
			&item.Temperatura, &item.SaturacionOxigeno, &item.PesoKg, &item.TallaCm,
			&item.DiagnosticoPrincipal, &item.DiagnosticoCIE10, &item.PlanManejo,
			&item.ProcedimientosIndicados, &item.ObservacionesMedico,
			&item.ProximaCita, &item.FechaConsulta, &item.EstadoConsulta,
			&nombre, &apellidos, &item.MedicoEspecialidad,
		); err != nil {
			log.Printf("scan consulta error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read consultas"})
			return
		}
		item.MedicoNombre = nombre + " " + apellidos
		consultas = append(consultas, item)
	}

	if err := rows.Err(); err != nil {
		log.Printf("rows error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch consultas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"consultas": consultas,
		"total":     len(consultas),
	})
}
