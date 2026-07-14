package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sinapsis-backend/models"
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
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// --- Visual style constants ---

var (
	// colorNavy fondo encabezados de sección
	colorNavy = &props.Color{Red: 30, Green: 42, Blue: 74}
	// colorTeal acento para sub-encabezados (sección de consulta)
	colorTeal = &props.Color{Red: 40, Green: 101, Blue: 128}
	// colorGray fondo muy suave para filas alternas y sub-bloques
	colorGray = &props.Color{Red: 248, Green: 249, Blue: 251}
	// colorGrayMid encabezados de tabla
	colorGrayMid = &props.Color{Red: 237, Green: 239, Blue: 244}
	// colorRed fondo etiqueta ANULADA
	colorRed = &props.Color{Red: 254, Green: 226, Blue: 226}
	// colorWhite texto sobre fondo oscuro
	colorWhite = &props.Color{Red: 255, Green: 255, Blue: 255}
	// colorBorder color de línea sutil para separadores
	colorBorder = &props.Color{Red: 210, Green: 214, Blue: 222}
)

// noBorder — celda sin ningún borde (layout limpio)
var noBorder = &props.Cell{}

// bottomBorder — solo línea inferior, separa filas sin peso visual
var bottomBorder = &props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}

func sectionHeaderCell() *props.Cell {
	return &props.Cell{
		BackgroundColor: colorNavy,
	}
}

func consultaHeaderCell() *props.Cell {
	return &props.Cell{
		BackgroundColor: colorTeal,
	}
}

func tableHeaderCell() *props.Cell {
	return &props.Cell{
		BorderType:      border.Bottom,
		BorderColor:     colorBorder,
		BackgroundColor: colorGrayMid,
	}
}

func medicoInfoCell() *props.Cell {
	return &props.Cell{
		BackgroundColor: colorGray,
	}
}

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

type pdfPacienteData struct {
	NombrePaciente    string
	ApellidosPaciente string
	TipoDocumento     string
	NumeroDocumento   string
	FechaNacimiento   time.Time
	Sexo              *string
	TipoSangre        *string
	Aseguradora       *string
	NumeroAfiliacion  *string
	NombreEntidad     string
	GeneradoEn        time.Time
	Consultas         []pdfConsultaData
	FormulasHuerfanas []pdfFormulaData
	AnexosHuerfanos   []pdfAnexoData
}

type pdfConsultaData struct {
	ID                      string
	FechaConsulta           time.Time
	TipoConsulta            *string
	MotivoConsulta          string
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
	MedicoNombreCompleto    string
	MedicoEspecialidad      string
	Formulas                []pdfFormulaData
	Anexos                  []pdfAnexoData
}

type pdfFormulaData struct {
	ID                string
	Medicamentos      []models.Medicamento
	Indicaciones      *string
	FechaPrescripcion time.Time
	EstadoFormula     string
}

type pdfAnexoData struct {
	TipoExamen  string
	Descripcion *string
	FechaCarga  time.Time
}

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

func formatBogota(t time.Time) string {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("02/01/2006 15:04")
}

// sanitizeFilename reemplaza caracteres inválidos en nombres de archivo por guiones bajos.
func sanitizeFilename(s string) string {
	result := make([]rune, 0, len([]rune(s)))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			result = append(result, r)
		case r == ' ':
			result = append(result, '_')
		default:
			result = append(result, '_')
		}
	}
	return string(result)
}

func buildFilename(nombreEntidad, nombrePaciente string) string {
	return fmt.Sprintf("historia_%s_%s.pdf",
		sanitizeFilename(nombreEntidad),
		sanitizeFilename(nombrePaciente),
	)
}

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
			if idx, ok := consultaIdx[f.ConsultaID.String()]; ok {
				consultas[idx].Formulas = append(consultas[idx].Formulas, fd)
				continue
			}
		}
		formulasHuerfanas = append(formulasHuerfanas, fd)
	}

	var anexosHuerfanos []pdfAnexoData
	for _, a := range anexos {
		ad := pdfAnexoData{TipoExamen: a.TipoExamen, Descripcion: a.Descripcion, FechaCarga: a.FechaCarga}
		if a.ConsultaID != nil {
			if idx, ok := consultaIdx[a.ConsultaID.String()]; ok {
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

// addSectionHeader agrega encabezado de sección: fondo navy, sin borde, texto blanco en negrita.
func addSectionHeader(m core.Maroto, title string) {
	m.AddRows(
		row.New(9).Add(
			text.NewCol(12, title, props.Text{
				Size:  10,
				Style: fontstyle.Bold,
				Align: align.Left,
				Top:   2.5,
				Left:  3,
				Color: colorWhite,
			}),
		).WithStyle(sectionHeaderCell()),
	)
}

// addConsultaHeader usa fondo teal para distinguir visualmente cada consulta.
func addConsultaHeader(m core.Maroto, title string) {
	m.AddRows(
		row.New(9).Add(
			text.NewCol(12, title, props.Text{
				Size:  9,
				Style: fontstyle.Bold,
				Align: align.Left,
				Top:   2.5,
				Left:  3,
				Color: colorWhite,
			}),
		).WithStyle(consultaHeaderCell()),
	)
}

// addFieldRow agrega una fila label/valor con solo línea inferior — sin cajas pesadas.
func addFieldRow(m core.Maroto, label, value string) {
	m.AddRows(
		row.New(calcFieldRowHeight(value, 8)).Add(
			text.NewCol(4, label, props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Left: 2, Color: &props.Color{Red: 100, Green: 110, Blue: 130}}),
			text.NewCol(8, value, props.Text{Size: 8, Top: 1.5}),
		).WithStyle(bottomBorder),
	)
}

// calcFieldRowHeight calcula la altura mínima (en mm) que necesita una fila de campo.
//
// IMPORTANTE: gofpdf ignora los saltos de línea — los aplana como espacios.
// Por eso se normaliza el texto a una sola línea antes de calcular el wrap real.
//
// Calibración Helvetica 8pt en A4 (márgenes 10mm):
//   - Columna valor (8/12 de 190mm) ≈ 127mm
//   - Ancho promedio por carácter ≈ 1.40mm
//   - Altura de línea ≈ 3.5mm
//   - Padding de celda = 2mm
func calcFieldRowHeight(value string, fontSize float64) float64 {
	usableColWidthMM := 127.0
	charWidthMM := 1.40
	lineHeightMM := 3.5
	paddingMM := 2.0
	minHeightMM := 6.0

	_ = fontSize

	if value == "" {
		return minHeightMM
	}

	// gofpdf aplana los saltos de línea: usamos el texto plano para calcular
	// cuántas líneas de word-wrap genera realmente el motor PDF.
	flat := flattenNewlines(value)
	segLen := len([]rune(flat))
	if segLen == 0 {
		return minHeightMM
	}

	charsPerLine := int(usableColWidthMM / charWidthMM)
	if charsPerLine < 1 {
		charsPerLine = 1
	}

	lines := (segLen + charsPerLine - 1) / charsPerLine
	if lines < 1 {
		lines = 1
	}

	height := float64(lines)*lineHeightMM + paddingMM
	if height < minHeightMM {
		height = minHeightMM
	}
	return height
}

// flattenNewlines reemplaza \r\n y \n por un espacio, imitando el comportamiento
// de gofpdf al renderizar texto (los saltos de línea se ignoran en celdas).
func flattenNewlines(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
			result = append(result, ' ')
			i++
		} else if s[i] == '\n' {
			result = append(result, ' ')
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// addMedsTableHeader encabezado de tabla sin bordes laterales.
func addMedsTableHeader(m core.Maroto) {
	m.AddRows(
		row.New(6).Add(
			text.NewCol(4, "Medicamento", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: &props.Color{Red: 60, Green: 70, Blue: 90}}),
			text.NewCol(2, "Dosis", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: &props.Color{Red: 60, Green: 70, Blue: 90}}),
			text.NewCol(3, "Frecuencia", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: &props.Color{Red: 60, Green: 70, Blue: 90}}),
			text.NewCol(3, "Duración", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: &props.Color{Red: 60, Green: 70, Blue: 90}}),
		).WithStyle(tableHeaderCell()),
	)
}

// addAnexosTableHeader encabezado de tabla de anexos sin bordes laterales.
func addAnexosTableHeader(m core.Maroto) {
	m.AddRows(
		row.New(6).Add(
			text.NewCol(3, "Tipo de examen", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: &props.Color{Red: 60, Green: 70, Blue: 90}}),
			text.NewCol(6, "Descripción", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: &props.Color{Red: 60, Green: 70, Blue: 90}}),
			text.NewCol(3, "Fecha de carga", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: &props.Color{Red: 60, Green: 70, Blue: 90}}),
		).WithStyle(tableHeaderCell()),
	)
}

// addFormulaBlock renderiza una fórmula como bloque visual limpio.
func addFormulaBlock(m core.Maroto, formula pdfFormulaData) {
	titulo := fmt.Sprintf("Fórmula  ·  %s", formatBogota(formula.FechaPrescripcion))
	headerStyle := &props.Cell{BackgroundColor: colorGray}
	titleColor := &props.Color{Red: 40, Green: 101, Blue: 128}
	if formula.EstadoFormula == "anulada" {
		headerStyle = &props.Cell{BackgroundColor: colorRed}
		titleColor = &props.Color{Red: 160, Green: 40, Blue: 40}
		titulo += "   ANULADA"
	}
	m.AddRows(
		row.New(7).Add(
			text.NewCol(12, titulo, props.Text{Size: 8, Style: fontstyle.Bold, Top: 2, Left: 2, Color: titleColor}),
		).WithStyle(headerStyle),
	)

	if len(formula.Medicamentos) > 0 {
		addMedsTableHeader(m)
		for _, med := range formula.Medicamentos {
			rowH := calcFieldRowHeight(med.Nombre, 8)
			m.AddRows(
				row.New(rowH).Add(
					text.NewCol(4, med.Nombre, props.Text{Size: 8, Top: 1.5}),
					text.NewCol(2, med.Dosis, props.Text{Size: 8, Top: 1.5}),
					text.NewCol(3, med.Frecuencia, props.Text{Size: 8, Top: 1.5}),
					text.NewCol(3, med.Duracion, props.Text{Size: 8, Top: 1.5}),
				).WithStyle(bottomBorder),
			)
		}
	}

	if formula.Indicaciones != nil {
		addFieldRow(m, "Indicaciones:", *formula.Indicaciones)
	}
	m.AddRow(3)
}

// renderPDF genera los bytes del PDF con layout moderno y minimalista.
func renderPDF(data pdfPacienteData) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageNumber(props.PageNumber{
			Pattern: "{current} / {total}",
			Place:   props.Bottom,
		}).
		Build()

	m := maroto.New(cfg)

	nombreCompleto := data.NombrePaciente + " " + data.ApellidosPaciente
	colorLabel := &props.Color{Red: 100, Green: 110, Blue: 130}

	// === PORTADA ===
	// Banda navy superior
	m.AddRows(
		row.New(18).Add(
			text.NewCol(12, "SINAPSIS", props.Text{
				Size:  20,
				Style: fontstyle.Bold,
				Align: align.Center,
				Top:   5,
				Color: colorWhite,
			}),
		).WithStyle(sectionHeaderCell()),
	)
	// Subtítulo y nombre del paciente
	m.AddRow(4)
	m.AddRows(
		row.New(8).Add(
			text.NewCol(12, "Historia Clínica", props.Text{
				Size:  11,
				Style: fontstyle.Normal,
				Align: align.Center,
				Top:   1,
				Color: colorLabel,
			}),
		).WithStyle(noBorder),
	)
	m.AddRows(
		row.New(10).Add(
			text.NewCol(12, nombreCompleto, props.Text{
				Size:  15,
				Style: fontstyle.Bold,
				Align: align.Center,
				Top:   2,
				Color: &props.Color{Red: 30, Green: 42, Blue: 74},
			}),
		).WithStyle(noBorder),
	)
	m.AddRow(2)
	// Línea divisoria
	m.AddRows(
		row.New(1).Add(
			text.NewCol(12, "", props.Text{}),
		).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
	)
	m.AddRow(4)
	// Metadatos en dos columnas
	m.AddRows(
		row.New(6).Add(
			text.NewCol(2, "Documento", props.Text{Size: 8, Top: 1, Color: colorLabel}),
			text.NewCol(4, fmt.Sprintf("%s  %s", data.TipoDocumento, data.NumeroDocumento), props.Text{Size: 8, Style: fontstyle.Bold, Top: 1}),
			text.NewCol(2, "Entidad", props.Text{Size: 8, Top: 1, Color: colorLabel}),
			text.NewCol(4, data.NombreEntidad, props.Text{Size: 8, Style: fontstyle.Bold, Top: 1}),
		).WithStyle(noBorder),
	)
	m.AddRows(
		row.New(6).Add(
			text.NewCol(2, "Generado", props.Text{Size: 8, Top: 1, Color: colorLabel}),
			text.NewCol(10, formatBogota(data.GeneradoEn), props.Text{Size: 8, Top: 1}),
		).WithStyle(noBorder),
	)
	m.AddRow(8)

	// === DATOS DEL PACIENTE ===
	addSectionHeader(m, "DATOS DEL PACIENTE")
	m.AddRow(2)
	if data.FechaNacimiento.Year() > 1 {
		addFieldRow(m, "Fecha de nacimiento", data.FechaNacimiento.Format("02/01/2006"))
	}
	if data.Sexo != nil {
		addFieldRow(m, "Sexo", *data.Sexo)
	}
	if data.TipoSangre != nil {
		addFieldRow(m, "Tipo de sangre", *data.TipoSangre)
	}
	if data.Aseguradora != nil {
		addFieldRow(m, "Aseguradora", *data.Aseguradora)
	}
	if data.NumeroAfiliacion != nil {
		addFieldRow(m, "Número de afiliación", *data.NumeroAfiliacion)
	}
	m.AddRow(8)

	// === CONSULTAS ===
	if len(data.Consultas) == 0 {
		m.AddRows(
			row.New(10).Add(
				text.NewCol(12, "Sin consultas registradas.", props.Text{
					Size:  9,
					Style: fontstyle.Italic,
					Align: align.Center,
					Top:   3,
					Color: colorLabel,
				}),
			).WithStyle(noBorder),
		)
	} else {
		for idx, consulta := range data.Consultas {
			addConsultaHeader(m, fmt.Sprintf("CONSULTA %d  ·  %s", idx+1, formatBogota(consulta.FechaConsulta)))

			// Fila de médico
			medicoText := fmt.Sprintf("%s  ·  %s", consulta.MedicoNombreCompleto, consulta.MedicoEspecialidad)
			m.AddRows(
				row.New(calcFieldRowHeight(medicoText, 8)).Add(
					text.NewCol(12, medicoText, props.Text{Size: 8, Style: fontstyle.Italic, Top: 1.5, Left: 2, Color: colorLabel}),
				).WithStyle(medicoInfoCell()),
			)
			m.AddRow(2)

			// Campos clínicos
			addFieldRow(m, "Motivo de consulta", consulta.MotivoConsulta)
			if consulta.Anamnesis != nil {
				addFieldRow(m, "Anamnesis", *consulta.Anamnesis)
			}
			if consulta.RevisionSistemas != nil {
				addFieldRow(m, "Revisión por sistemas", *consulta.RevisionSistemas)
			}
			if consulta.ExamenFisico != nil {
				addFieldRow(m, "Examen físico", *consulta.ExamenFisico)
			}
			if consulta.HallazgosClinicos != nil {
				addFieldRow(m, "Hallazgos clínicos", *consulta.HallazgosClinicos)
			}
			if consulta.DiagnosticoPrincipal != nil {
				addFieldRow(m, "Diagnóstico", *consulta.DiagnosticoPrincipal)
			}
			if consulta.DiagnosticoCIE10 != nil {
				addFieldRow(m, "CIE-10", *consulta.DiagnosticoCIE10)
			}
			if consulta.PlanManejo != nil {
				addFieldRow(m, "Plan de manejo", *consulta.PlanManejo)
			}
			if consulta.ProcedimientosIndicados != nil {
				addFieldRow(m, "Procedimientos indicados", *consulta.ProcedimientosIndicados)
			}
			if consulta.ObservacionesMedico != nil {
				addFieldRow(m, "Observaciones", *consulta.ObservacionesMedico)
			}

			// Signos vitales
			hasVitales := consulta.PresionArterial != nil || consulta.FrecuenciaCardiaca != nil ||
				consulta.FrecuenciaRespiratoria != nil || consulta.Temperatura != nil ||
				consulta.SaturacionOxigeno != nil || consulta.PesoKg != nil || consulta.TallaCm != nil

			if hasVitales {
				m.AddRow(3)
				m.AddRows(
					row.New(6).Add(
						text.NewCol(12, "Signos vitales", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: colorLabel}),
					).WithStyle(tableHeaderCell()),
				)
				if consulta.PresionArterial != nil {
					addFieldRow(m, "Presión arterial", *consulta.PresionArterial)
				}
				if consulta.FrecuenciaCardiaca != nil {
					addFieldRow(m, "Frecuencia cardíaca", fmt.Sprintf("%d lpm", *consulta.FrecuenciaCardiaca))
				}
				if consulta.FrecuenciaRespiratoria != nil {
					addFieldRow(m, "Frecuencia respiratoria", fmt.Sprintf("%d rpm", *consulta.FrecuenciaRespiratoria))
				}
				if consulta.Temperatura != nil {
					addFieldRow(m, "Temperatura", fmt.Sprintf("%.1f °C", *consulta.Temperatura))
				}
				if consulta.SaturacionOxigeno != nil {
					addFieldRow(m, "Saturación O₂", fmt.Sprintf("%d%%", *consulta.SaturacionOxigeno))
				}
				if consulta.PesoKg != nil {
					addFieldRow(m, "Peso", fmt.Sprintf("%.1f kg", *consulta.PesoKg))
				}
				if consulta.TallaCm != nil {
					addFieldRow(m, "Talla", fmt.Sprintf("%.1f cm", *consulta.TallaCm))
				}
			}

			// Fórmulas
			if len(consulta.Formulas) > 0 {
				m.AddRow(4)
				m.AddRows(
					row.New(6).Add(
						text.NewCol(12, "Fórmulas médicas", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: colorLabel}),
					).WithStyle(tableHeaderCell()),
				)
				for _, formula := range consulta.Formulas {
					addFormulaBlock(m, formula)
				}
			}

			// Anexos
			if len(consulta.Anexos) > 0 {
				m.AddRow(4)
				m.AddRows(
					row.New(6).Add(
						text.NewCol(12, "Anexos", props.Text{Size: 8, Style: fontstyle.Bold, Top: 1.5, Color: colorLabel}),
					).WithStyle(tableHeaderCell()),
				)
				addAnexosTableHeader(m)
				for _, anexo := range consulta.Anexos {
					desc := ""
					if anexo.Descripcion != nil {
						desc = *anexo.Descripcion
					}
					rowH := calcFieldRowHeight(desc, 8)
					m.AddRows(
						row.New(rowH).Add(
							text.NewCol(3, anexo.TipoExamen, props.Text{Size: 8, Top: 1.5}),
							text.NewCol(6, desc, props.Text{Size: 8, Top: 1.5}),
							text.NewCol(3, formatBogota(anexo.FechaCarga), props.Text{Size: 8, Top: 1.5}),
						).WithStyle(bottomBorder),
					)
				}
			}

			m.AddRow(8)
		}
	}

	// === FÓRMULAS HUÉRFANAS ===
	if len(data.FormulasHuerfanas) > 0 {
		addSectionHeader(m, "FÓRMULAS SIN CONSULTA ASOCIADA")
		m.AddRow(2)
		for _, formula := range data.FormulasHuerfanas {
			addFormulaBlock(m, formula)
		}
		m.AddRow(6)
	}

	// === ANEXOS HUÉRFANOS ===
	if len(data.AnexosHuerfanos) > 0 {
		addSectionHeader(m, "ANEXOS SIN CONSULTA ASOCIADA")
		m.AddRow(2)
		addAnexosTableHeader(m)
		for _, anexo := range data.AnexosHuerfanos {
			desc := ""
			if anexo.Descripcion != nil {
				desc = *anexo.Descripcion
			}
			rowH := calcFieldRowHeight(desc, 8)
			m.AddRows(
				row.New(rowH).Add(
					text.NewCol(3, anexo.TipoExamen, props.Text{Size: 8, Top: 1.5}),
					text.NewCol(6, desc, props.Text{Size: 8, Top: 1.5}),
					text.NewCol(3, formatBogota(anexo.FechaCarga), props.Text{Size: 8, Top: 1.5}),
				).WithStyle(bottomBorder),
			)
		}
	}

	document, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("maroto generation failed: %w", err)
	}
	return document.GetBytes(), nil
}

// --- Handler method ---

func (h *HistoriaClinicaPDFHandler) ExportPDF(c *gin.Context) {
	pacienteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	ctx := context.Background()

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

	const q1 = `
SELECT p.nombre_paciente, p.apellidos_paciente,
       p.tipo_documento, p.numero_documento,
       p.fecha_nacimiento, p.sexo,
       p.tipo_sangre, p.aseguradora, p.numero_afiliacion,
       e.nombre_entidad
FROM paciente p
JOIN historia_clinica hc ON hc.paciente_id = p.id
JOIN entidad e ON e.id = hc.entidad_id
WHERE p.id = $1`

	var (
		nombrePaciente, apellidosPaciente string
		tipoDocumento, numeroDocumento    string
		fechaNacimiento                   time.Time
		sexo, tipoSangre                  *string
		aseguradora, numeroAfiliacion     *string
		nombreEntidad                     string
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

	const q2 = `
SELECT c.id, c.fecha_consulta, c.tipo_consulta, c.motivo_consulta,
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
ORDER BY c.fecha_consulta DESC`

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

	const q3 = `
SELECT f.id, f.consulta_id, f.medicamentos, f.indicaciones,
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

	const q4 = `
SELECT e.consulta_id, e.tipo_examen, e.descripcion, e.fecha_carga
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

	generadoEn := time.Now()
	pdfData, err := buildPDFData(
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

	pdfBytes, err := renderPDF(pdfData)
	if err != nil {
		log.Printf("export pdf error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	filename := buildFilename(nombreEntidad, nombrePaciente+" "+apellidosPaciente)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
