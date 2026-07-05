package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
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
