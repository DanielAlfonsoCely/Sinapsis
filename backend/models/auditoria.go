package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditOperation string

const (
	AuditCreate           AuditOperation = "crear"
	AuditUpdate           AuditOperation = "actualizar"
	AuditDelete           AuditOperation = "eliminar"
	AuditConsult          AuditOperation = "consultar"
	AuditExport           AuditOperation = "exportar"
	AuditChangePermission AuditOperation = "cambiar_permisos"
	AuditUseAI            AuditOperation = "usar_ia"
	AuditLogin            AuditOperation = "iniciar_sesion"
)

type ImportanceLevel string

const (
	//Gravedad del evento
	Critical    = "CRITICAL"
	High        = "HIGH"
	Warning     = "WARNING"
	Informative = "INFORMATIVE"
)

type AuditLogEntry struct {
	ID                uuid.UUID       `json:"id"`
	UsuarioID         uuid.UUID       `json:"usuario_id"`
	UsuarioNombre     string          `json:"usuario_nombre"`
	UsuarioEmail      string          `json:"usuario_email"`
	TipoOperacion     AuditOperation  `json:"tipo_operacion"`
	TablaAfectada     string          `json:"tabla_afectada"`
	RegistroID        *uuid.UUID      `json:"registro_id"`
	ValoresAnteriores json.RawMessage `json:"valores_anteriores"`
	ValoresNuevos     json.RawMessage `json:"valores_nuevos"`
	IPOrigen          *string         `json:"ip_origen"`
	Detalles          *string         `json:"detalles"`
	FechaOperacion    time.Time       `json:"fecha_operacion"`
	Gravedad          ImportanceLevel `json:"gravedad"`
}
