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

type FormulaHandler struct {
	service *services.FormulaService
}

func NewFormulaHandler(service *services.FormulaService) *FormulaHandler {
	return &FormulaHandler{service: service}
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
	medicoID, err := h.service.ResolveMedicoID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repositories.ErrSoloMedico) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede gestionar fórmulas"})
			return uuid.Nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return uuid.Nil, false
	}
	return medicoID, true
}

// actorID extrae el user_id crudo del contexto (para auditoría; puede diferir
// de medicoID, que es el id de la fila "medico", no de "usuario").
func actorIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

// Create maneja POST /api/v1/formulas (HU-06).
// El médico registra una fórmula ligada a una consulta del paciente.
func (h *FormulaHandler) Create(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
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

	res, err := h.service.Create(c.Request.Context(), actorID, medicoID, pacienteID, consultaID, req, fechaVencimiento)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrPacienteNoACargo):
			c.JSON(http.StatusForbidden, gin.H{"error": "el paciente no está bajo su cuidado"})
		case errors.Is(err, repositories.ErrConsultaNoPertenece):
			c.JSON(http.StatusBadRequest, gin.H{"error": "la consulta no pertenece al paciente"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create fórmula"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                 res.FormulaID,
		"paciente_id":        pacienteID,
		"consulta_id":        consultaID,
		"fecha_prescripcion": res.FechaPrescripcion,
	})
}

// ListByPaciente maneja GET /api/v1/pacientes/:id/formulas.
// Devuelve las fórmulas del paciente, de la más reciente a la más antigua.
//
// Requiere sesión autenticada (ver routes.go): consultar las fórmulas de un
// paciente queda auditado como 'consultar'.
func (h *FormulaHandler) ListByPaciente(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	pacienteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	formulas, err := h.service.ListByPaciente(c.Request.Context(), actorID, pacienteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fórmulas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"formulas": formulas, "total": len(formulas)})
}

// Anular maneja POST /api/v1/formulas/:id/anular.
// Marca la fórmula como no vigente. Solo aplica a fórmulas de pacientes del médico.
func (h *FormulaHandler) Anular(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	medicoID, ok := h.resolveMedico(c)
	if !ok {
		return
	}
	formulaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fórmula id"})
		return
	}

	if err := h.service.Anular(c.Request.Context(), actorID, formulaID, medicoID); err != nil {
		if errors.Is(err, repositories.ErrFormulaNoAutorizada) {
			c.JSON(http.StatusForbidden, gin.H{"error": "la fórmula no corresponde a uno de sus pacientes"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to anular fórmula"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "anulada"})
}
