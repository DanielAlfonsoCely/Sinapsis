package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	maroto "github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"sinapsis-backend/models"
)

// HistoriaClinicaPDFHandler genera y entrega la historia clínica completa
// de un paciente como archivo PDF descargable.
type HistoriaClinicaPDFHandler struct {
	pool *pgxpool.Pool
}

// NewHistoriaClinicaPDFHandler crea una nueva instancia del handler con el pool de conexiones.
func NewHistoriaClinicaPDFHandler(pool *pgxpool.Pool) *HistoriaClinicaPDFHandler {
	return &HistoriaClinicaPDFHandler{pool: pool}
}

// --- Internal data structures ---

// pdfPacienteData contiene todos los datos necesarios para renderizar el PDF.
// Una vez poblado, no se realizan más llamadas a la base de datos.
type pdfPacienteData struct {
	// Identificación del paciente
	NombrePaciente   string
	ApellidosPaciente string
	TipoDocumento    string
	NumeroDocumento  string
	FechaNacimiento  time.Time
	Sexo             *string
	TipoSangre       *string
	Aseguradora      *string
	NumeroAfiliacion *string

	// Entidad médica
	NombreEntidad string

	// Marca de tiempo de generación (en America/Bogota)
	GeneradoEn time.Time

	// Datos clínicos
	Consultas []pdfConsultaData

	// Fórmulas y anexos no ligados a ninguna consulta
	FormulasHuerfanas []pdfFormulaData
	AnexosHuerfanos   []pdfAnexoData
}

// pdfConsultaData contiene los datos de una consulta médica para el PDF.
type pdfConsultaData struct {
	ID             string
	FechaConsulta  time.Time
	TipoConsulta   *string
	MotivoConsulta string

	// Campos clínicos opcionales (nil = omitir en el PDF)
	Anamnesis               *string
	RevisionSistemas        *string
	ExamenFisico            *string
	HallazgosClinicos       *string
	PresionArterial         *string
	FrecuenciaCardiaca      *int
	FrecuenciaRespiratoria  *int
	Temperatura             *float64
	SaturacionOxigeno       *int
	PesoKg                  *float64
	TallaCm                 *float64
	DiagnosticoPrincipal    *string
	DiagnosticoCIE10        *string
	PlanManejo              *string
	ProcedimientosIndicados *string
	ObservacionesMedico     *string

	// Médico que realizó la consulta
	MedicoNombreCompleto string
	MedicoEspecialidad   string

	// Registros asociados
	Formulas []pdfFormulaData
	Anexos   []pdfAnexoData
}

// pdfFormulaData contiene los datos de una fórmula médica para el PDF.
type pdfFormulaData struct {
	ID                string
	Medicamentos      []models.Medicamento
	Indicaciones      *string
	FechaPrescripcion time.Time
	EstadoFormula     string // "vigente" | "anulada"
}

// pdfAnexoData contiene los datos de un anexo para el PDF.
type pdfAnexoData struct {
	TipoExamen  string
	Descripcion *string
	FechaCarga  time.Time
}

// --- Raw row types (used during DB scanning) ---

type rawFormulaRow struct {
	ID                uuid.UUID
	ConsultaID        *uuid.UUID
	MedicamentosJSON  []byte
	Indicaciones      *string
	FechaPrescripcion time.Time
	EstadoFormula     string
}

type rawAnexoRow struct {
	ConsultaID  *uuid.UUID
	TipoExamen  string
	Descripcion *string
	FechaCarga  time.Time
}

// --- Helper functions ---

// formatBogota formatea un time.Time en la zona horaria America/Bogota
// con el formato DD/MM/YYYY HH:mm requerido por el requisito 2.8.
func formatBogota(t time.Time) string {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		// Si la zona horaria no está disponible, usar UTC como fallback seguro
		loc = time.UTC
	}
	return t.In(loc).Format("02/01/2006 15:04")
}

// buildFilename construye el nombre del archivo PDF con el formato:
// historia_clinica_<numero_documento>_<YYYYMMDD>.pdf
// donde la fecha está en la zona horaria America/Bogota (requisito 3.1).
func buildFilename(doc string, t time.Time) string {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		loc = time.UTC
	}
	dateStr := t.In(loc).Format("20060102")
	return fmt.Sprintf("historia_clinica_%s_%s.pdf", doc, dateStr)
}

// buildPDFData ensambla el struct pdfPacienteData completo a partir de los datos
// crudos leídos de la base de datos, asignando fórmulas y anexos a sus consultas
// o a las listas huérfanas según corresponda.
func buildPDFData(
	nombre, apellidos, tipoDoc, numDoc string,
	fechaNac time.Time,
	sexo, tipoSangre, aseguradora, numAfiliacion *string,
	nombreEntidad string,
	generadoEn time.Time,
	consultas []pdfConsultaData,
	formulas []rawFormulaRow,
	anexos []rawAnexoRow,
) (pdfPacienteData, error) {
	// Mapa de consulta UUID string → índice en el slice de consultas
	consultaIdx := make(map[string]int, len(consultas))
	for i, c := range consultas {
		consultaIdx[c.ID] = i
	}

	var formulasHuerfanas []pdfFormulaData
	for _, f := range formulas {
		meds := make([]models.Medicamento, 0)
		if err := json.Unmarshal(f.MedicamentosJSON, &meds); err != nil {
			return pdfPacienteData{}, fmt.Errorf("unmarshal medicamentos formula %s: %w", f.ID, err)
		}
		fd := pdfFormulaData{
			ID:                f.ID.String(),
			Medicamentos:      meds,
			Indicaciones:      f.Indicaciones,
			FechaPrescripcion: f.FechaPrescripcion,
			EstadoFormula:     f.EstadoFormula,
		}
		if f.ConsultaID != nil {
			key := f.ConsultaID.String()
			if idx, ok := consultaIdx[key]; ok {
				consultas[idx].Formulas = append(consultas[idx].Formulas, fd)
				continue
			}
		}
		// consulta_id nulo o consulta no encontrada → huérfana
		formulasHuerfanas = append(formulasHuerfanas, fd)
	}

	var anexosHuerfanos []pdfAnexoData
	for _, a := range anexos {
		ad := pdfAnexoData{
			TipoExamen:  a.TipoExamen,
			Descripcion: a.Descripcion,
			FechaCarga:  a.FechaCarga,
		}
		if a.ConsultaID != nil {
			key := a.ConsultaID.String()
			if idx, ok := consultaIdx[key]; ok {
				consultas[idx].Anexos = append(consultas[idx].Anexos, ad)
				continue
			}
		}
		anexosHuerfanos = append(anexosHuerfanos, ad)
	}

	if formulasHuerfanas == nil {
		formulasHuerfanas = []pdfFormulaData{}
	}
	if anexosHuerfanos == nil {
		anexosHuerfanos = []pdfAnexoData{}
	}

	return pdfPacienteData{
		NombrePaciente:    nombre,
		ApellidosPaciente: apellidos,
		TipoDocumento:     tipoDoc,
		NumeroDocumento:   numDoc,
		FechaNacimiento:   fechaNac,
		Sexo:              sexo,
		TipoSangre:        tipoSangre,
		Aseguradora:       aseguradora,
		NumeroAfiliacion:  numAfiliacion,
		NombreEntidad:     nombreEntidad,
		GeneradoEn:        generadoEn,
		Consultas:         consultas,
		FormulasHuerfanas: formulasHuerfanas,
		AnexosHuerfanos:   anexosHuerfanos,
	}, nil
}

// renderPDF genera los bytes del PDF a partir del struct pdfPacienteData.
func renderPDF(data pdfPacienteData) ([]byte, error) {
	// Configurar maroto con page numbering en el footer
	cfg := config.NewBuilder().
		WithPageNumber(props.PageNumber{
			Pattern: "Página {current} de {total}",
			Place:   props.Bottom,
		}).
		Build()

	m := maroto.New(cfg)

	// --- Cover Page ---
	m.AddRow(15, text.NewCol(12, "Historia Clínica — Sinapsis", props.Text{
		Size:  16,
		Style: fontstyle.Bold,
		Align: align.Center,
		Top:   5,
	}))

	nombreCompleto := data.NombrePaciente + " " + data.ApellidosPaciente
	m.AddRow(10, text.NewCol(12, nombreCompleto, props.Text{
		Size:  14,
		Style: fontstyle.Bold,
		Align: align.Center,
	}))

	m.AddRow(6, text.NewCol(12, fmt.Sprintf("%s — %s", data.TipoDocumento, data.NumeroDocumento), props.Text{
		Size:  11,
		Align: align.Center,
	}))

	m.AddRow(6, text.NewCol(12, data.NombreEntidad, props.Text{
		Size:  11,
		Align: align.Center,
	}))

	m.AddRow(8, text.NewCol(12, fmt.Sprintf("Generado: %s", formatBogota(data.GeneradoEn)), props.Text{
		Size:  10,
		Align: align.Center,
		Top:   2,
	}))

	// Espacio antes de la sección de datos del paciente
	m.AddRow(10)

	// --- Patient Data Section ---
	m.AddRow(8, text.NewCol(12, "Datos del Paciente", props.Text{
		Size:  12,
		Style: fontstyle.Bold,
		Top:   3,
	}))

	// Renderizar campos no nulos en tabla simple
	if data.FechaNacimiento.Year() > 1 {
		m.AddRow(5, 
			text.NewCol(4, "Fecha de Nacimiento:", props.Text{Size: 9, Style: fontstyle.Bold}),
			text.NewCol(8, formatBogota(data.FechaNacimiento), props.Text{Size: 9}),
		)
	}
	if data.Sexo != nil {
		m.AddRow(5,
			text.NewCol(4, "Sexo:", props.Text{Size: 9, Style: fontstyle.Bold}),
			text.NewCol(8, *data.Sexo, props.Text{Size: 9}),
		)
	}
	if data.TipoSangre != nil {
		m.AddRow(5,
			text.NewCol(4, "Tipo de Sangre:", props.Text{Size: 9, Style: fontstyle.Bold}),
			text.NewCol(8, *data.TipoSangre, props.Text{Size: 9}),
		)
	}
	if data.Aseguradora != nil {
		m.AddRow(5,
			text.NewCol(4, "Aseguradora:", props.Text{Size: 9, Style: fontstyle.Bold}),
			text.NewCol(8, *data.Aseguradora, props.Text{Size: 9}),
		)
	}
	if data.NumeroAfiliacion != nil {
		m.AddRow(5,
			text.NewCol(4, "Número de Afiliación:", props.Text{Size: 9, Style: fontstyle.Bold}),
			text.NewCol(8, *data.NumeroAfiliacion, props.Text{Size: 9}),
		)
	}

	// Espacio antes de consultas
	m.AddRow(8)

	// --- Consultas Section ---
	if len(data.Consultas) == 0 {
		m.AddRow(10, text.NewCol(12, "Sin consultas registradas", props.Text{
			Size:  11,
			Style: fontstyle.Italic,
			Align: align.Center,
		}))
	} else {
		for idx, consulta := range data.Consultas {
			// Encabezado de consulta
			m.AddRow(8, text.NewCol(12, fmt.Sprintf("Consulta %d — %s", idx+1, formatBogota(consulta.FechaConsulta)), props.Text{
				Size:  11,
				Style: fontstyle.Bold,
				Top:   2,
			}))

			// Médico
			m.AddRow(5, text.NewCol(12, fmt.Sprintf("Médico: %s | %s", consulta.MedicoNombreCompleto, consulta.MedicoEspecialidad), props.Text{
				Size:  9,
				Style: fontstyle.Italic,
			}))

			// Motivo de consulta (obligatorio)
			m.AddRow(5,
				text.NewCol(4, "Motivo de consulta:", props.Text{Size: 9, Style: fontstyle.Bold}),
				text.NewCol(8, consulta.MotivoConsulta, props.Text{Size: 9}),
			)

			// Campos opcionales
			if consulta.Anamnesis != nil {
				m.AddRow(5,
					text.NewCol(4, "Anamnesis:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.Anamnesis, props.Text{Size: 9}),
				)
			}
			if consulta.RevisionSistemas != nil {
				m.AddRow(5,
					text.NewCol(4, "Revisión por sistemas:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.RevisionSistemas, props.Text{Size: 9}),
				)
			}

			// Signos vitales (solo si al menos uno está presente)
			hasSignosVitales := consulta.PresionArterial != nil ||
				consulta.FrecuenciaCardiaca != nil ||
				consulta.FrecuenciaRespiratoria != nil ||
				consulta.Temperatura != nil ||
				consulta.SaturacionOxigeno != nil ||
				consulta.PesoKg != nil ||
				consulta.TallaCm != nil

			if hasSignosVitales {
				m.AddRow(5, text.NewCol(12, "Signos vitales:", props.Text{Size: 9, Style: fontstyle.Bold}))
				if consulta.PresionArterial != nil {
					m.AddRow(4,
						text.NewCol(5, "  Presión arterial:", props.Text{Size: 8}),
						text.NewCol(7, *consulta.PresionArterial, props.Text{Size: 8}),
					)
				}
				if consulta.FrecuenciaCardiaca != nil {
					m.AddRow(4,
						text.NewCol(5, "  Frecuencia cardíaca:", props.Text{Size: 8}),
						text.NewCol(7, fmt.Sprintf("%d lpm", *consulta.FrecuenciaCardiaca), props.Text{Size: 8}),
					)
				}
				if consulta.FrecuenciaRespiratoria != nil {
					m.AddRow(4,
						text.NewCol(5, "  Frecuencia respiratoria:", props.Text{Size: 8}),
						text.NewCol(7, fmt.Sprintf("%d rpm", *consulta.FrecuenciaRespiratoria), props.Text{Size: 8}),
					)
				}
				if consulta.Temperatura != nil {
					m.AddRow(4,
						text.NewCol(5, "  Temperatura:", props.Text{Size: 8}),
						text.NewCol(7, fmt.Sprintf("%.1f °C", *consulta.Temperatura), props.Text{Size: 8}),
					)
				}
				if consulta.SaturacionOxigeno != nil {
					m.AddRow(4,
						text.NewCol(5, "  Saturación de oxígeno:", props.Text{Size: 8}),
						text.NewCol(7, fmt.Sprintf("%d%%", *consulta.SaturacionOxigeno), props.Text{Size: 8}),
					)
				}
				if consulta.PesoKg != nil {
					m.AddRow(4,
						text.NewCol(5, "  Peso:", props.Text{Size: 8}),
						text.NewCol(7, fmt.Sprintf("%.1f kg", *consulta.PesoKg), props.Text{Size: 8}),
					)
				}
				if consulta.TallaCm != nil {
					m.AddRow(4,
						text.NewCol(5, "  Talla:", props.Text{Size: 8}),
						text.NewCol(7, fmt.Sprintf("%.1f cm", *consulta.TallaCm), props.Text{Size: 8}),
					)
				}
			}

			if consulta.ExamenFisico != nil {
				m.AddRow(5,
					text.NewCol(4, "Examen físico:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.ExamenFisico, props.Text{Size: 9}),
				)
			}
			if consulta.HallazgosClinicos != nil {
				m.AddRow(5,
					text.NewCol(4, "Hallazgos clínicos:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.HallazgosClinicos, props.Text{Size: 9}),
				)
			}
			if consulta.DiagnosticoPrincipal != nil {
				m.AddRow(5,
					text.NewCol(4, "Diagnóstico:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.DiagnosticoPrincipal, props.Text{Size: 9}),
				)
			}
			if consulta.DiagnosticoCIE10 != nil {
				m.AddRow(5,
					text.NewCol(4, "CIE-10:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.DiagnosticoCIE10, props.Text{Size: 9}),
				)
			}
			if consulta.PlanManejo != nil {
				m.AddRow(5,
					text.NewCol(4, "Plan de manejo:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.PlanManejo, props.Text{Size: 9}),
				)
			}
			if consulta.ProcedimientosIndicados != nil {
				m.AddRow(5,
					text.NewCol(4, "Procedimientos indicados:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.ProcedimientosIndicados, props.Text{Size: 9}),
				)
			}
			if consulta.ObservacionesMedico != nil {
				m.AddRow(5,
					text.NewCol(4, "Observaciones del médico:", props.Text{Size: 9, Style: fontstyle.Bold}),
					text.NewCol(8, *consulta.ObservacionesMedico, props.Text{Size: 9}),
				)
			}

			// Fórmulas de esta consulta
			if len(consulta.Formulas) > 0 {
				m.AddRow(6, text.NewCol(12, "Fórmulas:", props.Text{Size: 10, Style: fontstyle.Bold, Top: 2}))
				for _, formula := range consulta.Formulas {
					// Encabezado de fórmula
					titulo := fmt.Sprintf("Fórmula — %s", formatBogota(formula.FechaPrescripcion))
					if formula.EstadoFormula == "anulada" {
						titulo += " >> ANULADA <<"
					}
					m.AddRow(5, text.NewCol(12, titulo, props.Text{Size: 9, Style: fontstyle.Bold}))

					// Tabla de medicamentos
					if len(formula.Medicamentos) > 0 {
						// Encabezado de tabla
						headerRow := row.New(4).Add(
							text.NewCol(4, "Medicamento", props.Text{Size: 8, Style: fontstyle.Bold}),
							text.NewCol(2, "Dosis", props.Text{Size: 8, Style: fontstyle.Bold}),
							text.NewCol(3, "Frecuencia", props.Text{Size: 8, Style: fontstyle.Bold}),
							text.NewCol(3, "Duración", props.Text{Size: 8, Style: fontstyle.Bold}),
						).WithStyle(&props.Cell{BackgroundColor: &props.Color{Red: 240, Green: 240, Blue: 240}})
						m.AddRows(headerRow)

						for _, med := range formula.Medicamentos {
							m.AddRow(4,
								text.NewCol(4, med.Nombre, props.Text{Size: 8}),
								text.NewCol(2, med.Dosis, props.Text{Size: 8}),
								text.NewCol(3, med.Frecuencia, props.Text{Size: 8}),
								text.NewCol(3, med.Duracion, props.Text{Size: 8}),
							)
						}
					}

					// Indicaciones
					if formula.Indicaciones != nil {
						m.AddRow(4,
							text.NewCol(3, "Indicaciones:", props.Text{Size: 8, Style: fontstyle.Bold}),
							text.NewCol(9, *formula.Indicaciones, props.Text{Size: 8}),
						)
					}
				}
			}

			// Anexos de esta consulta
			if len(consulta.Anexos) > 0 {
				m.AddRow(6, text.NewCol(12, "Anexos:", props.Text{Size: 10, Style: fontstyle.Bold, Top: 2}))

				// Encabezado de tabla
				headerRow := row.New(4).Add(
					text.NewCol(3, "Tipo", props.Text{Size: 8, Style: fontstyle.Bold}),
					text.NewCol(6, "Descripción", props.Text{Size: 8, Style: fontstyle.Bold}),
					text.NewCol(3, "Fecha", props.Text{Size: 8, Style: fontstyle.Bold}),
				).WithStyle(&props.Cell{BackgroundColor: &props.Color{Red: 240, Green: 240, Blue: 240}})
				m.AddRows(headerRow)

				for _, anexo := range consulta.Anexos {
					desc := ""
					if anexo.Descripcion != nil {
						desc = *anexo.Descripcion
					}
					m.AddRow(4,
						text.NewCol(3, anexo.TipoExamen, props.Text{Size: 8}),
						text.NewCol(6, desc, props.Text{Size: 8}),
						text.NewCol(3, formatBogota(anexo.FechaCarga), props.Text{Size: 8}),
					)
				}
			}

			// Espacio entre consultas
			m.AddRow(6)
		}
	}

	// --- Fórmulas Huérfanas ---
	if len(data.FormulasHuerfanas) > 0 {
		m.AddRow(8, text.NewCol(12, "Fórmulas sin consulta asociada", props.Text{
			Size:  12,
			Style: fontstyle.Bold,
			Top:   3,
		}))

		for _, formula := range data.FormulasHuerfanas {
			titulo := fmt.Sprintf("Fórmula — %s", formatBogota(formula.FechaPrescripcion))
			if formula.EstadoFormula == "anulada" {
				titulo += " >> ANULADA <<"
			}
			m.AddRow(5, text.NewCol(12, titulo, props.Text{Size: 9, Style: fontstyle.Bold}))

			if len(formula.Medicamentos) > 0 {
				headerRow := row.New(4).Add(
					text.NewCol(4, "Medicamento", props.Text{Size: 8, Style: fontstyle.Bold}),
					text.NewCol(2, "Dosis", props.Text{Size: 8, Style: fontstyle.Bold}),
					text.NewCol(3, "Frecuencia", props.Text{Size: 8, Style: fontstyle.Bold}),
					text.NewCol(3, "Duración", props.Text{Size: 8, Style: fontstyle.Bold}),
				).WithStyle(&props.Cell{BackgroundColor: &props.Color{Red: 240, Green: 240, Blue: 240}})
				m.AddRows(headerRow)

				for _, med := range formula.Medicamentos {
					m.AddRow(4,
						text.NewCol(4, med.Nombre, props.Text{Size: 8}),
						text.NewCol(2, med.Dosis, props.Text{Size: 8}),
						text.NewCol(3, med.Frecuencia, props.Text{Size: 8}),
						text.NewCol(3, med.Duracion, props.Text{Size: 8}),
					)
				}
			}

			if formula.Indicaciones != nil {
				m.AddRow(4,
					text.NewCol(3, "Indicaciones:", props.Text{Size: 8, Style: fontstyle.Bold}),
					text.NewCol(9, *formula.Indicaciones, props.Text{Size: 8}),
				)
			}
		}

		m.AddRow(6)
	}

	// --- Anexos Huérfanos ---
	if len(data.AnexosHuerfanos) > 0 {
		m.AddRow(8, text.NewCol(12, "Anexos sin consulta asociada", props.Text{
			Size:  12,
			Style: fontstyle.Bold,
			Top:   3,
		}))

		headerRow := row.New(4).Add(
			text.NewCol(3, "Tipo", props.Text{Size: 8, Style: fontstyle.Bold}),
			text.NewCol(6, "Descripción", props.Text{Size: 8, Style: fontstyle.Bold}),
			text.NewCol(3, "Fecha", props.Text{Size: 8, Style: fontstyle.Bold}),
		).WithStyle(&props.Cell{BackgroundColor: &props.Color{Red: 240, Green: 240, Blue: 240}})
		m.AddRows(headerRow)

		for _, anexo := range data.AnexosHuerfanos {
			desc := ""
			if anexo.Descripcion != nil {
				desc = *anexo.Descripcion
			}
			m.AddRow(4,
				text.NewCol(3, anexo.TipoExamen, props.Text{Size: 8}),
				text.NewCol(6, desc, props.Text{Size: 8}),
				text.NewCol(3, formatBogota(anexo.FechaCarga), props.Text{Size: 8}),
			)
		}
	}

	// Generar el PDF
	document, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("maroto generation failed: %w", err)
	}

	return document.GetBytes(), nil
}

// --- Handler methods ---

// ExportPDF maneja GET /api/v1/pacientes/:id/historia-clinica/pdf
// Requiere autenticación JWT (RequireAuth middleware). El acceso al paciente
// ya fue validado por la UI; aquí solo verificamos que la historia exista.
func (h *HistoriaClinicaPDFHandler) ExportPDF(c *gin.Context) {
	// 1. Parsear :id como UUID
	pacienteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	ctx := context.Background()

	// 2. Verificar que la historia clínica existe y obtener su ID
	var hcID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT id FROM historia_clinica WHERE paciente_id = $1`, pacienteID,
	).Scan(&hcID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "historia clínica no encontrada"})
			return
		}
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	// 4. Query 1 — paciente + entidad
	const q1 = `
SELECT
    p.nombre_paciente, p.apellidos_paciente,
    p.tipo_documento, p.numero_documento,
    p.fecha_nacimiento, p.sexo,
    p.tipo_sangre, p.aseguradora, p.numero_afiliacion,
    e.nombre_entidad
FROM paciente p
JOIN historia_clinica hc ON hc.paciente_id = p.id
JOIN entidad e ON e.id = hc.entidad_id
WHERE p.id = $1`

	var (
		nombrePaciente   string
		apellidosPaciente string
		tipoDocumento    string
		numeroDocumento  string
		fechaNacimiento  time.Time
		sexo             *string
		tipoSangre       *string
		aseguradora      *string
		numeroAfiliacion *string
		nombreEntidad    string
	)
	err = h.pool.QueryRow(ctx, q1, pacienteID).Scan(
		&nombrePaciente, &apellidosPaciente,
		&tipoDocumento, &numeroDocumento,
		&fechaNacimiento, &sexo,
		&tipoSangre, &aseguradora, &numeroAfiliacion,
		&nombreEntidad,
	)
	if err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	// 5. Query 2 — consultas ASC
	const q2 = `
SELECT
    c.id, c.fecha_consulta, c.tipo_consulta, c.motivo_consulta,
    c.anamnesis, c.revision_sistemas, c.examen_fisico, c.hallazgos_clinicos,
    c.presion_arterial, c.frecuencia_cardiaca, c.frecuencia_respiratoria,
    c.temperatura, c.saturacion_oxigeno, c.peso_kg, c.talla_cm,
    c.diagnostico_principal, c.diagnostico_cie10, c.plan_manejo,
    c.procedimientos_indicados, c.observaciones_medico,
    u.nombre_usuario || ' ' || u.apellidos AS medico_nombre,
    m.especialidad AS medico_especialidad
FROM consulta c
JOIN medico m ON m.id = c.medico_id
JOIN usuario u ON u.id = m.usuario_id
WHERE c.historia_clinica_id = $1
ORDER BY c.fecha_consulta ASC`

	rows2, err := h.pool.Query(ctx, q2, hcID)
	if err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}
	defer rows2.Close()

	consultas := make([]pdfConsultaData, 0)
	for rows2.Next() {
		var cd pdfConsultaData
		var cid uuid.UUID
		var tempDec *float64
		if err := rows2.Scan(
			&cid, &cd.FechaConsulta, &cd.TipoConsulta, &cd.MotivoConsulta,
			&cd.Anamnesis, &cd.RevisionSistemas, &cd.ExamenFisico, &cd.HallazgosClinicos,
			&cd.PresionArterial, &cd.FrecuenciaCardiaca, &cd.FrecuenciaRespiratoria,
			&tempDec, &cd.SaturacionOxigeno, &cd.PesoKg, &cd.TallaCm,
			&cd.DiagnosticoPrincipal, &cd.DiagnosticoCIE10, &cd.PlanManejo,
			&cd.ProcedimientosIndicados, &cd.ObservacionesMedico,
			&cd.MedicoNombreCompleto, &cd.MedicoEspecialidad,
		); err != nil {
			log.Printf("export pdf error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
			return
		}
		cd.ID = cid.String()
		cd.Temperatura = tempDec
		cd.Formulas = []pdfFormulaData{}
		cd.Anexos = []pdfAnexoData{}
		consultas = append(consultas, cd)
	}
	if err := rows2.Err(); err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	// 6. Query 3 — fórmulas
	const q3 = `
SELECT
    f.id, f.consulta_id, f.medicamentos, f.indicaciones,
    f.fecha_prescripcion, f.estado_formula
FROM formula_medica f
WHERE f.historia_clinica_id = $1
ORDER BY f.fecha_prescripcion ASC`

	rows3, err := h.pool.Query(ctx, q3, hcID)
	if err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}
	defer rows3.Close()

	formulas := make([]rawFormulaRow, 0)
	for rows3.Next() {
		var fr rawFormulaRow
		if err := rows3.Scan(
			&fr.ID, &fr.ConsultaID, &fr.MedicamentosJSON, &fr.Indicaciones,
			&fr.FechaPrescripcion, &fr.EstadoFormula,
		); err != nil {
			log.Printf("export pdf error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
			return
		}
		formulas = append(formulas, fr)
	}
	if err := rows3.Err(); err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	// 7. Query 4 — anexos
	const q4 = `
SELECT
    e.consulta_id, e.tipo_examen, e.descripcion, e.fecha_carga
FROM examinagen e
WHERE e.historia_clinica_id = $1
ORDER BY e.fecha_carga ASC`

	rows4, err := h.pool.Query(ctx, q4, hcID)
	if err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}
	defer rows4.Close()

	anexos := make([]rawAnexoRow, 0)
	for rows4.Next() {
		var ar rawAnexoRow
		if err := rows4.Scan(
			&ar.ConsultaID, &ar.TipoExamen, &ar.Descripcion, &ar.FechaCarga,
		); err != nil {
			log.Printf("export pdf error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
			return
		}
		anexos = append(anexos, ar)
	}
	if err := rows4.Err(); err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	// 8. Ensamblar el struct completo
	generadoEn := time.Now()
	data, err := buildPDFData(
		nombrePaciente, apellidosPaciente,
		tipoDocumento, numeroDocumento,
		fechaNacimiento,
		sexo, tipoSangre, aseguradora, numeroAfiliacion,
		nombreEntidad,
		generadoEn,
		consultas, formulas, anexos,
	)
	if err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	// 9. Renderizar el PDF (stub en Tarea 2; implementación completa en Tarea 3)
	pdfBytes, err := renderPDF(data)
	if err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	// 10. Construir nombre de archivo y entregar la respuesta
	filename := buildFilename(numeroDocumento, generadoEn)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
