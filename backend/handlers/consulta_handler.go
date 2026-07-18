package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
	"sinapsis-backend/services"
)

type ConsultaHandler struct {
	service *services.ConsultaService
}

func NewConsultaHandler(service *services.ConsultaService) *ConsultaHandler {
	return &ConsultaHandler{service: service}
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

	ctx := c.Request.Context()

	medicoID, err := h.service.ResolveMedicoID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrSoloMedico) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede registrar consultas"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	res, err := h.service.Create(ctx, userID, medicoID, req, pacienteID, proximaCita)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrSinCitaHoy):
			c.JSON(http.StatusForbidden, gin.H{"error": "el paciente no tiene una cita activa para hoy"})
		case errors.Is(err, repositories.ErrHistoriaNoExiste):
			c.JSON(http.StatusNotFound, gin.H{"error": "el paciente no tiene historia clínica"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create consulta"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                  res.ConsultaID,
		"historia_clinica_id": res.HistoriaClinicaID,
		"paciente_id":         res.PacienteID,
		"fecha_consulta":      res.FechaConsulta,
		"formula_id":          res.FormulaID,
	})
}

// UpdatePreDiagnostico maneja PATCH /api/v1/consultas/:id/pre-diagnostico.
//
// RF-12/RN-007: el médico debe registrar su propia impresión clínica (pre-
// diagnóstico) ANTES de poder solicitar o visualizar hallazgos de análisis IA
// sobre los exámenes de esta consulta. Solo el médico que atendió la consulta
// puede registrarlo.
func (h *ConsultaHandler) UpdatePreDiagnostico(c *gin.Context) {
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

	consultaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid consulta id"})
		return
	}

	var req models.UpdatePreDiagnosticoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	medicoID, err := h.service.ResolveMedicoID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrSoloMedico) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede registrar el pre-diagnóstico"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	if err := h.service.UpdatePreDiagnostico(ctx, userID, consultaID, medicoID, req.PreDiagnostico); err != nil {
		if errors.Is(err, repositories.ErrConsultaNoAutorizada) {
			c.JSON(http.StatusNotFound, gin.H{"error": "consulta no encontrada o no autorizada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update pre-diagnóstico"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": consultaID, "pre_diagnostico": req.PreDiagnostico})
}

// ListByPaciente maneja GET /api/v1/pacientes/:id/consultas (HU-04).
// Devuelve el historial de consultas del paciente, de la más reciente a la más
// antigua, incluyendo el médico que atendió cada consulta.
//
// Requiere sesión autenticada (ver routes.go): la lectura del historial
// clínico completo de un paciente queda auditada como 'consultar'.
func (h *ConsultaHandler) ListByPaciente(c *gin.Context) {
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

	pacienteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	consultas, err := h.service.ListByPaciente(c.Request.Context(), userID, pacienteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch consultas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"consultas": consultas,
		"total":     len(consultas),
	})
}
