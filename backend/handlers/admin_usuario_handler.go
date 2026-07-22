package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
)

// rolesValidos contiene los valores permitidos para el parámetro `rol`.
var rolesValidos = map[string]bool{
	"medico":           true,
	"paciente":         true,
	"admin_entidad":    true,
	"admin_plataforma": true,
}

type AdminUsuarioHandler struct {
	pool *pgxpool.Pool
}

func NewAdminUsuarioHandler(pool *pgxpool.Pool) *AdminUsuarioHandler {
	return &AdminUsuarioHandler{pool: pool}
}

// ListUsuarios maneja GET /api/v1/admin/usuarios
// Parámetros de query: q, rol, entidad_id, limit (default 20, rango [1,100]), offset (default 0)
func (h *AdminUsuarioHandler) ListUsuarios(c *gin.Context) {
	// --- 1. Parsear y validar limit ---
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit debe estar entre 1 y 100"})
		return
	}

	// --- 2. Parsear offset ---
	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// --- 3. Parsear y validar rol ---
	rol := c.Query("rol")
	if rol != "" && !rolesValidos[rol] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("rol inválido: %s", rol)})
		return
	}

	// --- 4. Parsear q y entidad_id ---
	q := c.Query("q")

	entidadIDStr := c.Query("entidad_id")
	var entidadID pgtype.UUID
	if entidadIDStr != "" {
		if err := entidadID.Scan(entidadIDStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "entidad_id inválido"})
			return
		}
	}
	// Si entidadIDStr está vacío, entidadID queda con Valid=false (nil UUID)

	ctx := context.Background()

	// --- 5. Query principal con LEFT JOINs ---
	const queryLista = `
SELECT
    u.id,
    u.nombre_usuario,
    u.apellidos,
    u.email,
    u.tipo_usuario,
    u.estado,
    u.fecha_actualizacion,
    CASE
        WHEN u.tipo_usuario = 'medico'        THEN em.nombre_entidad
        WHEN u.tipo_usuario = 'admin_entidad' THEN ea.nombre_entidad
        WHEN u.tipo_usuario = 'paciente'      THEN ehc.nombre_entidad
        ELSE NULL
    END AS entidad_nombre
FROM usuario u
LEFT JOIN medico                med ON med.usuario_id = u.id
LEFT JOIN entidad               em  ON em.id = med.entidad_id
LEFT JOIN administrador_entidad ae  ON ae.usuario_id  = u.id
LEFT JOIN entidad               ea  ON ea.id = ae.entidad_id
LEFT JOIN paciente              p   ON p.usuario_id   = u.id
LEFT JOIN historia_clinica      hc  ON hc.paciente_id = p.id
LEFT JOIN entidad               ehc ON ehc.id = hc.entidad_id
WHERE
    (NULLIF($1, '') IS NULL OR (
        u.nombre_usuario   ILIKE '%' || $1 || '%' OR
        u.apellidos        ILIKE '%' || $1 || '%' OR
        u.email            ILIKE '%' || $1 || '%' OR
        p.numero_documento ILIKE '%' || $1 || '%'
    ))
    AND (NULLIF($2, '') IS NULL OR u.tipo_usuario = NULLIF($2, '')::tipo_usuario_enum)
    AND ($3::uuid IS NULL OR med.entidad_id = $3 OR ae.entidad_id = $3 OR hc.entidad_id = $3)
ORDER BY u.fecha_actualizacion DESC
LIMIT  $4
OFFSET $5`

	rows, err := h.pool.Query(ctx, queryLista, q, rol, entidadID, limit, offset)
	if err != nil {
		log.Printf("list usuarios query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch usuarios"})
		return
	}
	defer rows.Close()

	usuarios := make([]models.AdminUsuarioItem, 0)
	for rows.Next() {
		var u models.AdminUsuarioItem
		if err := rows.Scan(
			&u.ID,
			&u.NombreUsuario,
			&u.Apellidos,
			&u.Email,
			&u.TipoUsuario,
			&u.Estado,
			&u.FechaActualizacion,
			&u.EntidadNombre,
		); err != nil {
			log.Printf("scan usuario error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read usuarios"})
			return
		}
		usuarios = append(usuarios, u)
	}
	if err := rows.Err(); err != nil {
		log.Printf("rows error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch usuarios"})
		return
	}

	// --- 6. Query de conteos ---
	const queryConteos = `
SELECT
    COUNT(*)                                     AS total,
    COUNT(*) FILTER (WHERE u.estado = true)      AS total_activos,
    COUNT(*) FILTER (WHERE u.estado = false)     AS total_inactivos
FROM usuario u
LEFT JOIN medico                med ON med.usuario_id = u.id
LEFT JOIN administrador_entidad ae  ON ae.usuario_id  = u.id
LEFT JOIN paciente              p   ON p.usuario_id   = u.id
LEFT JOIN historia_clinica      hc  ON hc.paciente_id = p.id
WHERE
    (NULLIF($1, '') IS NULL OR (
        u.nombre_usuario   ILIKE '%' || $1 || '%' OR
        u.apellidos        ILIKE '%' || $1 || '%' OR
        u.email            ILIKE '%' || $1 || '%' OR
        p.numero_documento ILIKE '%' || $1 || '%'
    ))
    AND (NULLIF($2, '') IS NULL OR u.tipo_usuario = NULLIF($2, '')::tipo_usuario_enum)
    AND ($3::uuid IS NULL OR med.entidad_id = $3 OR ae.entidad_id = $3 OR hc.entidad_id = $3)`

	var total, totalActivos, totalInactivos int
	err = h.pool.QueryRow(ctx, queryConteos, q, rol, entidadID).Scan(
		&total,
		&totalActivos,
		&totalInactivos,
	)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("conteos query error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch conteos"})
			return
		}
		// Sin filas → todos los conteos quedan en 0
	}

	// --- 7. Responder 200 ---
	c.JSON(http.StatusOK, models.ListUsuariosResponse{
		Usuarios:       usuarios,
		Total:          total,
		TotalActivos:   totalActivos,
		TotalInactivos: totalInactivos,
		Limit:          limit,
		Offset:         offset,
	})
}

// PatchRol maneja PATCH /api/v1/admin/usuarios/:id/rol
// Actualiza el tipo_usuario de un usuario y devuelve los campos actualizados.
func (h *AdminUsuarioHandler) PatchRol(c *gin.Context) {
	// --- 1. Parsear :id como UUID ---
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	// --- 2. Bind PatchRolRequest (gin valida oneof automáticamente) ---
	var req models.PatchRolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// --- 3. Ejecutar UPDATE ... RETURNING ---
	const query = `
UPDATE usuario
SET
    tipo_usuario        = $1::tipo_usuario_enum,
    fecha_actualizacion = NOW()
WHERE id = $2
RETURNING id, tipo_usuario, fecha_actualizacion`

	var resp models.PatchRolResponse
	err = h.pool.QueryRow(context.Background(), query, req.TipoUsuario, id).Scan(
		&resp.ID,
		&resp.TipoUsuario,
		&resp.FechaActualizacion,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
			return
		}
		log.Printf("patch rol query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rol"})
		return
	}

	// --- 4. Responder 200 con PatchRolResponse ---
	c.JSON(http.StatusOK, resp)
}
