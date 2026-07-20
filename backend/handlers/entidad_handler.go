package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
)

type EntidadHandler struct {
	pool *pgxpool.Pool
}

func NewEntidadHandler(pool *pgxpool.Pool) *EntidadHandler {
	return &EntidadHandler{pool: pool}
}

// Create maneja POST /api/v1/entidades.
// Registra una nueva entidad de salud. Solo accesible por admin.
func (h *EntidadHandler) Create(c *gin.Context) {
	var req models.CreateEntidadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var entidad models.Entidad
	err := h.pool.QueryRow(
		context.Background(),
		`INSERT INTO entidad (nombre_entidad, tipo_entidad, nit, ciudad, direccion, telefono, estado, fecha_creacion)
		 VALUES ($1, $2, $3, $4, $5, $6, true, NOW())
		 RETURNING id, nombre_entidad, tipo_entidad, nit, direccion, telefono, ciudad, estado, fecha_creacion`,
		req.NombreEntidad, req.TipoEntidad, req.NIT, req.Ciudad, req.Direccion, req.Telefono,
	).Scan(
		&entidad.ID, &entidad.NombreEntidad, &entidad.TipoEntidad, &entidad.NIT,
		&entidad.Direccion, &entidad.Telefono, &entidad.Ciudad, &entidad.Estado, &entidad.FechaCreacion,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "ya existe una entidad con ese NIT"})
			return
		}
		log.Printf("create entidad error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create entidad"})
		return
	}

	c.JSON(http.StatusCreated, entidad)
}

// List maneja GET /api/v1/entidades.
func (h *EntidadHandler) List(c *gin.Context) {
	rows, err := h.pool.Query(
		context.Background(),
		`SELECT id, nombre_entidad, tipo_entidad, nit, direccion, telefono, ciudad, estado, fecha_creacion
		 FROM entidad
		 ORDER BY nombre_entidad`,
	)
	if err != nil {
		log.Printf("list entidades error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidades"})
		return
	}
	defer rows.Close()

	entidades := make([]models.Entidad, 0)
	for rows.Next() {
		var e models.Entidad
		if err := rows.Scan(
			&e.ID, &e.NombreEntidad, &e.TipoEntidad, &e.NIT,
			&e.Direccion, &e.Telefono, &e.Ciudad, &e.Estado, &e.FechaCreacion,
		); err != nil {
			log.Printf("scan entidad error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read entidades"})
			return
		}
		entidades = append(entidades, e)
	}

	c.JSON(http.StatusOK, gin.H{"entidades": entidades, "total": len(entidades)})
}

// ListAdmin maneja GET /api/v1/admin/entidades.
// Solo accesible por admin_plataforma (protegido por RequireAdmin middleware).
func (h *EntidadHandler) ListAdmin(c *gin.Context) {
	q := c.DefaultQuery("q", "")

	rows, err := h.pool.Query(
		context.Background(),
		`SELECT id, nombre_entidad, tipo_entidad, nit, ciudad, estado, fecha_creacion
		 FROM entidad
		 WHERE (
		     $1 = ''
		     OR nombre_entidad ILIKE '%' || $1 || '%'
		     OR nit            ILIKE '%' || $1 || '%'
		     OR ciudad         ILIKE '%' || $1 || '%'
		 )
		 ORDER BY nombre_entidad`,
		q,
	)
	if err != nil {
		log.Printf("list admin entidades error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidades"})
		return
	}
	defer rows.Close()

	entidades := make([]models.AdminEntidadListItem, 0)
	for rows.Next() {
		var e models.AdminEntidadListItem
		if err := rows.Scan(
			&e.ID, &e.NombreEntidad, &e.TipoEntidad, &e.NIT,
			&e.Ciudad, &e.Estado, &e.FechaCreacion,
		); err != nil {
			log.Printf("scan admin entidad error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read entidades"})
			return
		}
		entidades = append(entidades, e)
	}

	c.JSON(http.StatusOK, models.AdminEntidadListResponse{
		Entidades: entidades,
		Total:     len(entidades),
	})
}

// GetByIDAdmin maneja GET /api/v1/admin/entidades/:id.
// Solo accesible por admin_plataforma (protegido por RequireAdmin middleware).
func (h *EntidadHandler) GetByIDAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	// Query 1: datos base de la entidad
	var detalle models.EntidadDetalle
	err = h.pool.QueryRow(
		context.Background(),
		`SELECT id, nombre_entidad, tipo_entidad, nit, ciudad, direccion, telefono, estado, fecha_creacion
		 FROM entidad WHERE id = $1`,
		id,
	).Scan(
		&detalle.ID, &detalle.NombreEntidad, &detalle.TipoEntidad, &detalle.NIT,
		&detalle.Ciudad, &detalle.Direccion, &detalle.Telefono, &detalle.Estado, &detalle.FechaCreacion,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "entidad no encontrada"})
			return
		}
		log.Printf("get entidad by id error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidad"})
		return
	}

	// Query 2: convenios activos (historias clínicas de la entidad)
	err = h.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM historia_clinica WHERE entidad_id = $1`,
		id,
	).Scan(&detalle.ConveniosActivos)
	if err != nil {
		log.Printf("get convenios activos error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch convenios"})
		return
	}

	// Query 3: períodos históricos
	periodoRows, err := h.pool.Query(
		context.Background(),
		`SELECT EXTRACT(YEAR FROM fecha_creacion)::int AS anio,
		        COUNT(*)::int                          AS cantidad_historias
		 FROM historia_clinica
		 WHERE entidad_id = $1
		 GROUP BY anio
		 ORDER BY anio DESC`,
		id,
	)
	if err != nil {
		log.Printf("get periodos historicos error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch periodos"})
		return
	}
	defer periodoRows.Close()

	detalle.PeriodosHistoricos = make([]models.PeriodoHistorico, 0)
	for periodoRows.Next() {
		var p models.PeriodoHistorico
		if err := periodoRows.Scan(&p.Anio, &p.CantidadHistorias); err != nil {
			log.Printf("scan periodo error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read periodos"})
			return
		}
		detalle.PeriodosHistoricos = append(detalle.PeriodosHistoricos, p)
	}

	// Query 4: usuarios asociados
	usuarioRows, err := h.pool.Query(
		context.Background(),
		`SELECT u.id, u.nombre_usuario, u.apellidos, u.tipo_usuario
		 FROM usuario u
		 WHERE u.id IN (
		     SELECT usuario_id FROM medico WHERE entidad_id = $1
		     UNION
		     SELECT usuario_id FROM administrador_entidad WHERE entidad_id = $1
		 )
		 ORDER BY u.apellidos, u.nombre_usuario`,
		id,
	)
	if err != nil {
		log.Printf("get usuarios asociados error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch usuarios"})
		return
	}
	defer usuarioRows.Close()

	detalle.UsuariosAsociados = make([]models.UsuarioAsociado, 0)
	for usuarioRows.Next() {
		var u models.UsuarioAsociado
		if err := usuarioRows.Scan(&u.ID, &u.NombreUsuario, &u.Apellidos, &u.TipoUsuario); err != nil {
			log.Printf("scan usuario error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read usuarios"})
			return
		}
		detalle.UsuariosAsociados = append(detalle.UsuariosAsociados, u)
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
	var totalConsultas, totalPacientesActivos, totalUsuariosActivos int

	err := h.pool.QueryRow(
		context.Background(),
		`SELECT
		    (SELECT COUNT(*) FROM consulta)                          AS total_consultas,
		    (SELECT COUNT(*) FROM paciente WHERE estado = true)      AS total_pacientes_activos,
		    (SELECT COUNT(*) FROM usuario  WHERE estado = true)      AS total_usuarios_activos`,
	).Scan(&totalConsultas, &totalPacientesActivos, &totalUsuariosActivos)
	if err != nil {
		log.Printf("stats query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_consultas":         totalConsultas,
		"total_pacientes_activos": totalPacientesActivos,
		"total_usuarios_activos":  totalUsuariosActivos,
	})
}
