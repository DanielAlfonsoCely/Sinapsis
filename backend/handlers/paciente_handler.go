package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
)

type PacienteHandler struct {
	pool *pgxpool.Pool
}

func NewPacienteHandler(pool *pgxpool.Pool) *PacienteHandler {
	return &PacienteHandler{pool: pool}
}

// List maneja GET /api/v1/pacientes?q=texto
// q es opcional: filtra por nombre, apellidos o número de documento.
func (h *PacienteHandler) List(c *gin.Context) {
	q := c.Query("q")

	rows, err := h.pool.Query(
		context.Background(),
		`SELECT p.id, p.numero_documento, p.tipo_documento, p.nombre_paciente, p.apellidos_paciente,
		        p.telefono, p.email,
		        (SELECT MAX(co.fecha_consulta) FROM consulta co WHERE co.paciente_id = p.id) AS ultima_consulta,
		        p.estado
		 FROM paciente p
		 WHERE $1 = '' 
		    OR p.nombre_paciente ILIKE '%' || $1 || '%'
		    OR p.apellidos_paciente ILIKE '%' || $1 || '%'
		    OR p.numero_documento ILIKE '%' || $1 || '%'
		 ORDER BY p.nombre_paciente, p.apellidos_paciente`,
		q,
	)
	if err != nil {
		log.Printf("list pacientes error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pacientes"})
		return
	}
	defer rows.Close()

	pacientes := make([]models.PacienteListItem, 0)
	for rows.Next() {
		var p models.PacienteListItem
		if err := rows.Scan(
			&p.ID, &p.NumeroDocumento, &p.TipoDocumento, &p.NombrePaciente, &p.ApellidosPaciente,
			&p.Telefono, &p.Email, &p.UltimaConsulta, &p.Estado,
		); err != nil {
			log.Printf("scan paciente error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read pacientes"})
			return
		}
		// proxima_cita: aún no existe módulo de agenda, se deja siempre en null
		p.ProximaCita = nil
		pacientes = append(pacientes, p)
	}

	if err := rows.Err(); err != nil {
		log.Printf("rows error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pacientes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pacientes": pacientes,
		"total":     len(pacientes),
	})
}

// GetByID maneja GET /api/v1/pacientes/:id
func (h *PacienteHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	var p models.Paciente
	err = h.pool.QueryRow(
		context.Background(),
		`SELECT id, usuario_id, numero_documento, tipo_documento, nombre_paciente, apellidos_paciente,
		        fecha_nacimiento, sexo, tipo_sangre, alergias, direccion, telefono, email,
		        contacto_emergencia, telefono_emergencia, antecedentes_medicos, medicamentos_actuales,
		        estado_civil, ocupacion, aseguradora, numero_afiliacion, fecha_registro, estado
		 FROM paciente WHERE id = $1`,
		id,
	).Scan(
		&p.ID, &p.UsuarioID, &p.NumeroDocumento, &p.TipoDocumento, &p.NombrePaciente, &p.ApellidosPaciente,
		&p.FechaNacimiento, &p.Sexo, &p.TipoSangre, &p.Alergias, &p.Direccion, &p.Telefono, &p.Email,
		&p.ContactoEmergencia, &p.TelefonoEmergencia, &p.AntecedentesMedicos, &p.MedicamentosActuales,
		&p.EstadoCivil, &p.Ocupacion, &p.Aseguradora, &p.NumeroAfiliacion, &p.FechaRegistro, &p.Estado,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "paciente not found"})
			return
		}
		log.Printf("get paciente error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch paciente"})
		return
	}

	c.JSON(http.StatusOK, p)
}
