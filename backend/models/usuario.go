package models

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	NombreUsuario string `json:"nombre_usuario" binding:"required"`
	Apellidos     string `json:"apellidos" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	Contrasena    string `json:"contrasena" binding:"required,min=8"`
	TipoUsuario   string `json:"tipo_usuario" binding:"required,oneof=medico paciente admin_entidad admin_plataforma"`
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Contrasena string `json:"contrasena" binding:"required"`
}

type UpdateUserRequest struct {
	NombreUsuario *string `json:"nombre_usuario"`
	Apellidos     *string `json:"apellidos"`
	Email         *string `json:"email" binding:"omitempty,email"`
	TipoUsuario   *string `json:"tipo_usuario" binding:"omitempty,oneof=medico paciente admin_entidad admin_plataforma"`
	Estado        *bool   `json:"estado"`
}

type RoleRequest struct {
	TipoUsuario string `json:"tipo_usuario" binding:"required,oneof=medico paciente admin_entidad admin_plataforma"`
}

// CreateUsuarioAdminRequest es el cuerpo de POST /api/v1/admin/usuarios.
// Incluye los campos base de usuario más los campos específicos según
// tipo_usuario, que se validan a mano en el repositorio porque son
// condicionales (no todos aplican a todos los roles).
type CreateUsuarioAdminRequest struct {
	NombreUsuario string `json:"nombre_usuario" binding:"required"`
	Apellidos     string `json:"apellidos" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	Contrasena    string `json:"contrasena" binding:"required,min=8"`
	TipoUsuario   string `json:"tipo_usuario" binding:"required,oneof=medico paciente admin_entidad admin_plataforma"`

	// Médico
	NumeroDocumento  string `json:"numero_documento"`
	Especialidad     string `json:"especialidad"`
	NumeroColegiado  string `json:"numero_colegiado"`
	ExperienciaAnios *int   `json:"experiencia_anios"`
	EntidadID        string `json:"entidad_id"`

	// Paciente
	TipoDocumento   string `json:"tipo_documento"`
	FechaNacimiento string `json:"fecha_nacimiento"`
	Sexo            string `json:"sexo"`
	Telefono        string `json:"telefono"`

	// Admin Entidad usa el mismo EntidadID de arriba.
}

type Usuario struct {
	ID                 uuid.UUID `json:"id"`
	NombreUsuario      string    `json:"nombre_usuario"`
	Apellidos          string    `json:"apellidos"`
	Email              string    `json:"email"`
	Contrasena         string    `json:"-"`
	TipoUsuario        string    `json:"tipo_usuario"`
	Estado             bool      `json:"estado"`
	FechaCreacion      time.Time `json:"fecha_creacion"`
	FechaActualizacion time.Time `json:"fecha_actualizacion"`
}

type LoginResponse struct {
	Token   string `json:"token"`
	Usuario any    `json:"usuario"`
}

// AdminUsuarioItem representa cada fila de la tabla del dashboard admin.
type AdminUsuarioItem struct {
	ID                 uuid.UUID `json:"id"`
	NombreUsuario      string    `json:"nombre_usuario"`
	Apellidos          string    `json:"apellidos"`
	Email              string    `json:"email"`
	TipoUsuario        string    `json:"tipo_usuario"`
	Estado             bool      `json:"estado"`
	EntidadNombre      *string   `json:"entidad_nombre"`
	FechaActualizacion time.Time `json:"fecha_actualizacion"`
}

// ListUsuariosResponse es el cuerpo de respuesta de GET /api/v1/admin/usuarios.
type ListUsuariosResponse struct {
	Usuarios       []AdminUsuarioItem `json:"usuarios"`
	Total          int                `json:"total"`
	TotalActivos   int                `json:"total_activos"`
	TotalInactivos int                `json:"total_inactivos"`
	Limit          int                `json:"limit"`
	Offset         int                `json:"offset"`
}

// PatchRolRequest es el cuerpo de PATCH /api/v1/admin/usuarios/:id/rol.
type PatchRolRequest struct {
	TipoUsuario string `json:"tipo_usuario" binding:"required,oneof=medico paciente admin_entidad admin_plataforma"`
}

// PatchRolResponse es la respuesta de PATCH /api/v1/admin/usuarios/:id/rol.
type PatchRolResponse struct {
	ID                 uuid.UUID `json:"id"`
	TipoUsuario        string    `json:"tipo_usuario"`
	FechaActualizacion time.Time `json:"fecha_actualizacion"`
}