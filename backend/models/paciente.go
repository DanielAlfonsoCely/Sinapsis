package models

import (
	"time"

	"github.com/google/uuid"
)

// PacienteListItem es lo que se muestra en la tabla de /pacientes.
// UltimaConsulta y ProximaCita pueden ser nil si el paciente aún no tiene consultas.
type PacienteListItem struct {
	ID                uuid.UUID  `json:"id"`
	NumeroDocumento   string     `json:"numero_documento"`
	TipoDocumento     string     `json:"tipo_documento"`
	NombrePaciente    string     `json:"nombre_paciente"`
	ApellidosPaciente string     `json:"apellidos_paciente"`
	Telefono          *string    `json:"telefono"`
	Email             *string    `json:"email"`
	UltimaConsulta    *time.Time `json:"ultima_consulta"`
	ProximaCita       *time.Time `json:"proxima_cita"`
	TieneCitaHoy      bool       `json:"tiene_cita_hoy"` // hay cita 'programada' para hoy -> se puede consultar
	Estado            bool       `json:"estado"`
}

// CreatePacienteRequest es el payload para registrar un paciente nuevo (HU-02).
// Crea en una sola transacción: usuario (login del paciente), paciente y su
// historia_clinica (RN-003: un paciente, una historia clínica).
type CreatePacienteRequest struct {
	NumeroDocumento   string  `json:"numero_documento" binding:"required"`
	TipoDocumento     string  `json:"tipo_documento" binding:"required,oneof=CC TI CE PA RC"`
	NombrePaciente    string  `json:"nombre_paciente" binding:"required"`
	ApellidosPaciente string  `json:"apellidos_paciente" binding:"required"`
	FechaNacimiento   string  `json:"fecha_nacimiento" binding:"required"` // YYYY-MM-DD
	Sexo              *string `json:"sexo" binding:"omitempty,oneof=M F O"`
	Email             string  `json:"email" binding:"required,email"`
	Telefono          *string `json:"telefono"`
	Direccion         *string `json:"direccion"`
}

// TransferRequest es el payload para remitir/transferir un paciente a otro médico.
type TransferRequest struct {
	MedicoDestinoID string `json:"medico_destino_id" binding:"required,uuid"`
}

// MedicoListItem es cada médico disponible como destino de una remisión.
type MedicoListItem struct {
	ID           uuid.UUID `json:"id"`
	Nombre       string    `json:"nombre"`
	Especialidad string    `json:"especialidad"`
}

// Paciente es el detalle completo, tal como está en la tabla paciente.
type Paciente struct {
	ID                   uuid.UUID `json:"id"`
	UsuarioID            uuid.UUID `json:"usuario_id"`
	NumeroDocumento      string    `json:"numero_documento"`
	TipoDocumento        string    `json:"tipo_documento"`
	NombrePaciente       string    `json:"nombre_paciente"`
	ApellidosPaciente    string    `json:"apellidos_paciente"`
	FechaNacimiento      time.Time `json:"fecha_nacimiento"`
	Sexo                 *string   `json:"sexo"`
	TipoSangre           *string   `json:"tipo_sangre"`
	Alergias             *string   `json:"alergias"`
	Direccion            *string   `json:"direccion"`
	Telefono             *string   `json:"telefono"`
	Email                *string   `json:"email"`
	ContactoEmergencia   *string   `json:"contacto_emergencia"`
	TelefonoEmergencia   *string   `json:"telefono_emergencia"`
	AntecedentesMedicos  *string   `json:"antecedentes_medicos"`
	MedicamentosActuales *string   `json:"medicamentos_actuales"`
	EstadoCivil          *string   `json:"estado_civil"`
	Ocupacion            *string   `json:"ocupacion"`
	Aseguradora          *string   `json:"aseguradora"`
	NumeroAfiliacion     *string   `json:"numero_afiliacion"`
	FechaRegistro        time.Time `json:"fecha_registro"`
	Estado               bool      `json:"estado"`
}
