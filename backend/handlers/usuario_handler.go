package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sinapsis-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UsuarioHandler struct {
	pool *pgxpool.Pool
}

// NewUsuarioHandler crea el handler de usuarios con acceso al pool de BD.
func NewUsuarioHandler(pool *pgxpool.Pool) *UsuarioHandler {
	return &UsuarioHandler{pool: pool}
}

func (h *UsuarioHandler) CrearUsuario(c *gin.Context) {
	// 1. Verificar autenticación.
	if _, ok := c.Get("user_id"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No autenticado"})
		return
	}

	// 2. Verificar que sea admin_plataforma.
	userType, exists := c.Get("tipo_usuario")
	if !exists || userType != "admin_plataforma" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el administrador de plataforma puede crear usuarios"})
		return
	}

	// 3. Parsear el JSON.
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Hash de la contraseña.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	// 5. Insertar en la BD.
	ctx := context.Background()
	var id uuid.UUID
	var fechaCreacion time.Time
	err = h.pool.QueryRow(ctx,
		`INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario, estado, fecha_creacion, fecha_actualizacion)
		 VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
		 RETURNING id, fecha_creacion`,
		req.NombreUsuario, req.Apellidos, req.Email, string(hashedPassword), req.TipoUsuario,
	).Scan(&id, &fechaCreacion)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un usuario con ese email"})
			return
		}
		log.Printf("Error al crear usuario: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear el usuario"})
		return
	}

	// 6. Devolver respuesta sin la contraseña.
	c.JSON(http.StatusCreated, gin.H{
		"id":                  id,
		"nombre_usuario":      req.NombreUsuario,
		"apellidos":           req.Apellidos,
		"email":               req.Email,
		"tipo_usuario":        req.TipoUsuario,
		"fecha_creacion":      fechaCreacion,
		"fecha_actualizacion": fechaCreacion,
		"estado":              true,
	})
}

func (h *UsuarioHandler) EditarUsuario(c *gin.Context) {
	// 1. Verificar autenticación.
	if _, ok := c.Get("user_id"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No autenticado"})
		return
	}

	// 2. Verificar que sea admin_plataforma.
	userType, exists := c.Get("tipo_usuario")
	if !exists || userType != "admin_plataforma" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el administrador de plataforma puede editar usuarios"})
		return
	}

	// 3. Obtener ID del usuario a editar.
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
		return
	}

	// 4. Parsear body.
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 5. Construir query dinámica solo con los campos enviados.
	setParts := make([]string, 0, 5)
	args := make([]interface{}, 0, 5)
	argIndex := 1

	if req.NombreUsuario != nil {
		setParts = append(setParts, fmt.Sprintf("nombre_usuario = $%d", argIndex))
		args = append(args, *req.NombreUsuario)
		argIndex++
	}
	if req.Apellidos != nil {
		setParts = append(setParts, fmt.Sprintf("apellidos = $%d", argIndex))
		args = append(args, *req.Apellidos)
		argIndex++
	}
	if req.Email != nil {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, *req.Email)
		argIndex++
	}
	if req.TipoUsuario != nil {
		setParts = append(setParts, fmt.Sprintf("tipo_usuario = $%d", argIndex))
		args = append(args, *req.TipoUsuario)
		argIndex++
	}
	if req.Estado != nil {
		setParts = append(setParts, fmt.Sprintf("estado = $%d", argIndex))
		args = append(args, *req.Estado)
		argIndex++
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se enviaron campos para actualizar"})
		return
	}

	// 6. Ejecutar la actualización.
	setParts = append(setParts, "fecha_actualizacion = NOW()")
	query := `UPDATE usuario SET ` + strings.Join(setParts, ", ") + fmt.Sprintf(" WHERE id = $%d AND estado = true", argIndex)
	args = append(args, targetID)

	tag, err := h.pool.Exec(context.Background(), query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un usuario con ese email"})
			return
		}
		log.Printf("Error al editar usuario: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al editar el usuario"})
		return
	}

	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	// 7. Responder al cliente.
	c.JSON(http.StatusOK, gin.H{"mensaje": "Usuario actualizado correctamente"})
}

func (h *UsuarioHandler) EliminarUsuario(c *gin.Context) {
	// 1. Verificar autenticación.
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No autenticado"})
		return
	}

	// 2. Verificar que sea admin_plataforma.
	adminID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}

	// 3. Obtener ID del usuario a eliminar.
	userType, exists := c.Get("tipo_usuario")
	if !exists || userType != "admin_plataforma" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el administrador de plataforma puede eliminar usuarios"})
		return
	}

	// 4. No permitir que el admin se elimine a sí mismo.
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
		return
	}

	if targetID == adminID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No puedes eliminarte a ti mismo"})
		return
	}

	// 5. Desactivar usuario sin borrarlo físicamente.
	tag, err := h.pool.Exec(context.Background(),
		`UPDATE usuario SET estado = false, fecha_actualizacion = NOW()
		 WHERE id = $1 AND estado = true`,
		targetID,
	)
	if err != nil {
		log.Printf("Error al eliminar usuario: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el usuario"})
		return
	}

	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado o ya inactivo"})
		return
	}

	// 6. Responder al cliente.
	c.JSON(http.StatusOK, gin.H{"mensaje": "Usuario desactivado correctamente"})
}

func (h *UsuarioHandler) AsignarRol(c *gin.Context) {
	// 1. Verificar autenticación.
	if _, ok := c.Get("user_id"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No autenticado"})
		return
	}

	// 2. Verificar que sea admin_plataforma.
	userType, exists := c.Get("tipo_usuario")
	if !exists || userType != "admin_plataforma" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el administrador de plataforma puede asignar roles"})
		return
	}

	// 3. Obtener ID del usuario.
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
		return
	}

	// 4. Parsear body.
	var req models.RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 5. Actualizar el rol.
	tag, err := h.pool.Exec(context.Background(),
		`UPDATE usuario SET tipo_usuario = $1, fecha_actualizacion = NOW()
		 WHERE id = $2 AND estado = true`,
		req.TipoUsuario, targetID,
	)
	if err != nil {
		log.Printf("Error al asignar rol: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al asignar el rol"})
		return
	}

	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado o inactivo"})
		return
	}

	// 6. Responder al cliente.
	c.JSON(http.StatusOK, gin.H{"mensaje": "Rol asignado correctamente"})
}

func (h *UsuarioHandler) ObtenerUsuarios(c *gin.Context) {
	if _, ok := c.Get("user_id"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No autenticado"})
		return
	}

	userType, exists := c.Get("tipo_usuario")
	if !exists || userType != "admin_plataforma" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el administrador de plataforma puede ver usuarios"})
		return
	}

	search := c.Query("search")
	rol := c.Query("rol")
	estadoParam := c.Query("estado")
	limit := c.DefaultQuery("limit", "20")
	offset := c.DefaultQuery("offset", "0")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit inválido"})
		return
	}
	offsetInt, err := strconv.Atoi(offset)
	if err != nil || offsetInt < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset inválido"})
		return
	}

	query := `SELECT id, nombre_usuario, apellidos, email, tipo_usuario, estado, fecha_creacion, fecha_actualizacion FROM usuario WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM usuario WHERE 1=1`
	args := make([]interface{}, 0, 6)
	countArgs := make([]interface{}, 0, 6)
	argIndex := 1

	if search != "" {
		clause := fmt.Sprintf(` AND (nombre_usuario ILIKE $%d OR apellidos ILIKE $%d OR email ILIKE $%d)`, argIndex, argIndex+1, argIndex+2)
		query += clause
		countQuery += clause
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}

	if rol != "" {
		query += fmt.Sprintf(` AND tipo_usuario = $%d`, argIndex)
		countQuery += fmt.Sprintf(` AND tipo_usuario = $%d`, argIndex)
		args = append(args, rol)
		countArgs = append(countArgs, rol)
		argIndex++
	}

	if estadoParam != "" {
		estado, err := strconv.ParseBool(estadoParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "estado inválido, use true o false"})
			return
		}
		query += fmt.Sprintf(` AND estado = $%d`, argIndex)
		countQuery += fmt.Sprintf(` AND estado = $%d`, argIndex)
		args = append(args, estado)
		countArgs = append(countArgs, estado)
		argIndex++
	}

	query += ` ORDER BY fecha_creacion DESC`
	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIndex, argIndex+1)
	args = append(args, limitInt, offsetInt)

	rows, err := h.pool.Query(context.Background(), query, args...)
	if err != nil {
		log.Printf("Error al obtener usuarios: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener usuarios"})
		return
	}
	defer rows.Close()

	usuarios := make([]models.Usuario, 0)
	for rows.Next() {
		var u models.Usuario
		if err := rows.Scan(&u.ID, &u.NombreUsuario, &u.Apellidos, &u.Email, &u.TipoUsuario, &u.Estado, &u.FechaCreacion, &u.FechaActualizacion); err != nil {
			log.Printf("Error al escanear usuario: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer usuarios"})
			return
		}
		usuarios = append(usuarios, u)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterando usuarios: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer usuarios"})
		return
	}

	var total int
	if err := h.pool.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total); err != nil {
		log.Printf("Error al contar usuarios: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al contar usuarios"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"usuarios": usuarios,
		"total":    total,
		"limit":    limitInt,
		"offset":   offsetInt,
	})
}
