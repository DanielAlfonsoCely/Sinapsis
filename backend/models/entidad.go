package models

import (
	"time"

	"github.com/google/uuid"
)

type Entidad struct {
	ID            uuid.UUID `json:"id"`
	NombreEntidad string    `json:"nombre_entidad"`
	TipoEntidad   string    `json:"tipo_entidad"`
	NIT           string    `json:"nit"`
	Direccion     *string   `json:"direccion"`
	Telefono      *string   `json:"telefono"`
	Ciudad        *string   `json:"ciudad"`
	Estado        bool      `json:"estado"`
	FechaCreacion time.Time `json:"fecha_creacion"`
}

type CreateEntidadRequest struct {
	NombreEntidad string  `json:"nombre_entidad" binding:"required,max=150"`
	TipoEntidad   string  `json:"tipo_entidad" binding:"required,oneof=IPS EPS clinica hospital consultorio"`
	NIT           string  `json:"nit" binding:"required,max=50"`
	Ciudad        *string `json:"ciudad"`
	Direccion     *string `json:"direccion"`
	Telefono      *string `json:"telefono"`
}

// AdminEntidadListItem es cada elemento de GET /api/v1/admin/entidades.
type AdminEntidadListItem struct {
	ID            uuid.UUID `json:"id"`
	NombreEntidad string    `json:"nombre_entidad"`
	TipoEntidad   string    `json:"tipo_entidad"`
	NIT           string    `json:"nit"`
	Ciudad        *string   `json:"ciudad"`
	Estado        bool      `json:"estado"`
	FechaCreacion time.Time `json:"fecha_creacion"`
}

// AdminEntidadListResponse es el cuerpo de GET /api/v1/admin/entidades.
type AdminEntidadListResponse struct {
	Entidades []AdminEntidadListItem `json:"entidades"`
	Total     int                    `json:"total"`
}

// PeriodoHistorico es un elemento de periodos_historicos en el detalle.
type PeriodoHistorico struct {
	Anio              int `json:"anio"`
	CantidadHistorias int `json:"cantidad_historias"`
}

// UsuarioAsociado es un elemento de usuarios_asociados en el detalle.
type UsuarioAsociado struct {
	ID            uuid.UUID `json:"id"`
	NombreUsuario string    `json:"nombre_usuario"`
	Apellidos     string    `json:"apellidos"`
	TipoUsuario   string    `json:"tipo_usuario"`
}

// EntidadDetalle es el cuerpo de GET /api/v1/admin/entidades/:id.
type EntidadDetalle struct {
	ID                 uuid.UUID          `json:"id"`
	NombreEntidad      string             `json:"nombre_entidad"`
	TipoEntidad        string             `json:"tipo_entidad"`
	NIT                string             `json:"nit"`
	Ciudad             *string            `json:"ciudad"`
	Direccion          *string            `json:"direccion"`
	Telefono           *string            `json:"telefono"`
	Estado             bool               `json:"estado"`
	FechaCreacion      time.Time          `json:"fecha_creacion"`
	ConveniosActivos   int                `json:"convenios_activos"`
	PeriodosHistoricos []PeriodoHistorico `json:"periodos_historicos"`
	UsuariosAsociados  []UsuarioAsociado  `json:"usuarios_asociados"`
}
