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
