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
