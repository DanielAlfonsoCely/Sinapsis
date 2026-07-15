package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
	"sinapsis-backend/services"
)

type UsuarioHandler struct {
	service *services.UsuarioService
}

// NewUsuarioHandler crea el handler de usuarios apoyado en la capa de servicio.
func NewUsuarioHandler(service *services.UsuarioService) *UsuarioHandler {
	return &UsuarioHandler{service: service}
}

func (h *UsuarioHandler) CrearUsuario(c *gin.Context) {
	actorID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}

	var req models.CreateUsuarioAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.Create(c.Request.Context(), actorID, req)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrDuplicateEmail):
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un usuario con ese email"})
			return
		case errors.Is(err, repositories.ErrDuplicateDocumento):
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un registro con ese número de documento"})
			return
		case errors.Is(err, repositories.ErrEntidadRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "entidad_id es requerido para este rol"})
			return
		}
		log.Printf("Error al crear usuario: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear el usuario"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                  user.ID,
		"nombre_usuario":      user.NombreUsuario,
		"apellidos":           user.Apellidos,
		"email":               user.Email,
		"tipo_usuario":        user.TipoUsuario,
		"estado":              user.Estado,
		"fecha_creacion":      user.FechaCreacion,
		"fecha_actualizacion": user.FechaActualizacion,
	})
}

func (h *UsuarioHandler) EditarUsuario(c *gin.Context) {
	actorID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.Update(c.Request.Context(), targetID, actorID, req)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrNoUpdateFields):
			c.JSON(http.StatusBadRequest, gin.H{"error": "No se enviaron campos para actualizar"})
			return
		case errors.Is(err, repositories.ErrDuplicateEmail):
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un usuario con ese email"})
			return
		case errors.Is(err, repositories.ErrUsuarioNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			return
		}
		log.Printf("Error al editar usuario: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al editar el usuario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Usuario actualizado correctamente"})
}
func (h *UsuarioHandler) EliminarUsuario(c *gin.Context) {
	adminIDRaw := c.GetString("user_id")
	adminID, err := uuid.Parse(adminIDRaw)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
		return
	}

	err = h.service.Delete(c.Request.Context(), targetID, adminID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCannotDeleteSelf):
			c.JSON(http.StatusBadRequest, gin.H{"error": "No puedes eliminarte a ti mismo"})
			return
		case errors.Is(err, repositories.ErrUsuarioNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado o ya inactivo"})
			return
		}
		log.Printf("Error al eliminar usuario: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el usuario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Usuario desactivado correctamente"})
}

func (h *UsuarioHandler) AsignarRol(c *gin.Context) {
	actorID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario inválido"})
		return
	}

	var req models.RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.AssignRole(c.Request.Context(), targetID, actorID, req.TipoUsuario)
	if err != nil {
		if errors.Is(err, repositories.ErrUsuarioNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado o inactivo"})
			return
		}
		log.Printf("Error al asignar rol: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al asignar el rol"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Rol asignado correctamente"})
}

func (h *UsuarioHandler) ObtenerUsuarios(c *gin.Context) {
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

	var estadoFilter *bool
	if estadoParam != "" {
		estado, err := strconv.ParseBool(estadoParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "estado inválido, use true o false"})
			return
		}
		estadoFilter = &estado
	}

	usuarios, total, err := h.service.List(c.Request.Context(), repositories.UsuarioFilters{
		Search: search,
		Rol:    rol,
		Estado: estadoFilter,
		Limit:  limitInt,
		Offset: offsetInt,
	})
	if err != nil {
		log.Printf("Error al obtener usuarios: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener usuarios"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"usuarios": usuarios,
		"total":    total,
		"limit":    limitInt,
		"offset":   offsetInt,
	})
}