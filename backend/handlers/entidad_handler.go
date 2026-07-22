package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
	"sinapsis-backend/services"
)

type EntidadHandler struct {
	service *services.EntidadService
}

func NewEntidadHandler(service *services.EntidadService) *EntidadHandler {
	return &EntidadHandler{service: service}
}

// actorIDFromContext extrae el user_id crudo del contexto (para auditoría),
// igual que en FormulaHandler.
func (h *EntidadHandler) actorIDFromContext(c *gin.Context) (uuid.UUID, bool) {
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

// Create maneja POST /api/v1/entidades.
// Registra una nueva entidad de salud. Solo accesible por admin.
func (h *EntidadHandler) Create(c *gin.Context) {
	actorID, ok := h.actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	var req models.CreateEntidadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entidad, err := h.service.Create(c.Request.Context(), actorID, req)
	if err != nil {
		if errors.Is(err, repositories.ErrEntidadDuplicada) {
			c.JSON(http.StatusConflict, gin.H{"error": "ya existe una entidad con ese NIT"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create entidad"})
		return
	}

	c.JSON(http.StatusCreated, entidad)
}

// List maneja GET /api/v1/entidades.
func (h *EntidadHandler) List(c *gin.Context) {
	entidades, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidades"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entidades": entidades, "total": len(entidades)})
}

// ListAdmin maneja GET /api/v1/admin/entidades.
// Solo accesible por admin_plataforma (protegido por RequireAdmin middleware).
func (h *EntidadHandler) ListAdmin(c *gin.Context) {
	actorID, ok := h.actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	q := c.DefaultQuery("q", "")

	entidades, err := h.service.ListAdmin(c.Request.Context(), actorID, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidades"})
		return
	}

	c.JSON(http.StatusOK, models.AdminEntidadListResponse{
		Entidades: entidades,
		Total:     len(entidades),
	})
}

// GetByIDAdmin maneja GET /api/v1/admin/entidades/:id.
// Solo accesible por admin_plataforma (protegido por RequireAdmin middleware).
func (h *EntidadHandler) GetByIDAdmin(c *gin.Context) {
	actorID, ok := h.actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	detalle, err := h.service.GetByIDAdmin(c.Request.Context(), actorID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "entidad no encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidad"})
		return
	}

	c.JSON(http.StatusOK, detalle)
}

// ListPacientesAdmin maneja GET /api/v1/admin/entidades/:id/pacientes.
// Devuelve los pacientes con historia clínica registrada en esta entidad.
func (h *EntidadHandler) ListPacientesAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	// Verificar que la entidad existe
	var existe bool
	err = h.pool.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM entidad WHERE id = $1)`,
		id,
	).Scan(&existe)
	if err != nil || !existe {
		c.JSON(http.StatusNotFound, gin.H{"error": "entidad no encontrada"})
		return
	}

	rows, err := h.pool.Query(
		context.Background(),
		`SELECT
		    p.id,
		    p.numero_documento,
		    p.tipo_documento,
		    p.nombre_paciente,
		    p.apellidos_paciente,
		    p.telefono,
		    p.email,
		    p.estado,
		    p.fecha_registro,
		    MAX(c.fecha_consulta) AS ultima_consulta
		 FROM historia_clinica hc
		 JOIN paciente p ON p.id = hc.paciente_id
		 LEFT JOIN consulta c ON c.paciente_id = p.id
		 WHERE hc.entidad_id = $1
		 GROUP BY p.id
		 ORDER BY p.apellidos_paciente, p.nombre_paciente`,
		id,
	)
	if err != nil {
		log.Printf("list pacientes entidad error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pacientes"})
		return
	}
	defer rows.Close()

	pacientes := make([]models.PacienteEntidadItem, 0)
	for rows.Next() {
		var p models.PacienteEntidadItem
		if err := rows.Scan(
			&p.ID, &p.NumeroDocumento, &p.TipoDocumento,
			&p.NombrePaciente, &p.ApellidosPaciente,
			&p.Telefono, &p.Email, &p.Estado, &p.FechaRegistro,
			&p.UltimaConsulta,
		); err != nil {
			log.Printf("scan paciente entidad error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read pacientes"})
			return
		}
		pacientes = append(pacientes, p)
	}

	c.JSON(http.StatusOK, gin.H{"pacientes": pacientes, "total": len(pacientes)})
}

// UpdateAdmin maneja PUT /api/v1/admin/entidades/:id.
// Solo accesible por admin_plataforma.
func (h *EntidadHandler) UpdateAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req models.UpdateEntidadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var entidad models.Entidad
	err = h.pool.QueryRow(
		context.Background(),
		`UPDATE entidad
		 SET nombre_entidad = $1,
		     tipo_entidad   = $2,
		     nit            = $3,
		     ciudad         = $4,
		     direccion      = $5,
		     telefono       = $6,
		     estado         = $7
		 WHERE id = $8
		 RETURNING id, nombre_entidad, tipo_entidad, nit, direccion, telefono, ciudad, estado, fecha_creacion`,
		req.NombreEntidad, req.TipoEntidad, req.NIT,
		req.Ciudad, req.Direccion, req.Telefono, req.Estado,
		id,
	).Scan(
		&entidad.ID, &entidad.NombreEntidad, &entidad.TipoEntidad, &entidad.NIT,
		&entidad.Direccion, &entidad.Telefono, &entidad.Ciudad, &entidad.Estado, &entidad.FechaCreacion,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "entidad no encontrada"})
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "ya existe una entidad con ese NIT"})
			return
		}
		log.Printf("update entidad error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update entidad"})
		return
	}

	c.JSON(http.StatusOK, entidad)
}

// Stats maneja GET /api/v1/admin/stats.
// Devuelve métricas globales de la plataforma para el dashboard admin.
func (h *EntidadHandler) Stats(c *gin.Context) {
	actorID, ok := h.actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	stats, err := h.service.Stats(c.Request.Context(), actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_consultas":         stats.TotalConsultas,
		"total_pacientes_activos": stats.TotalPacientesActivos,
		"total_usuarios_activos":  stats.TotalUsuariosActivos,
	})
}
