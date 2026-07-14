package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnexoHandler struct {
	pool       *pgxpool.Pool
	uploadsDir string
}

func NewAnexoHandler(pool *pgxpool.Pool, uploadsDir string) *AnexoHandler {
	return &AnexoHandler{pool: pool, uploadsDir: uploadsDir}
}

// categoriaPorExt clasifica el anexo como imagen o documento para la UI.
func categoriaPorExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".dcm", ".nii", ".nii.gz":
		return "imagen"
	default:
		return "documento"
	}
}

// extensionCompuesta detecta extensiones de dos partes usadas por formatos de
// imagen médica (p.ej. "estudio.nii.gz" → ".nii.gz"). filepath.Ext solo
// captura el último componente (".gz"), lo que le hace perder la extensión
// ".nii" que MONAI/Nibabel necesitan para reconocer el formato NIfTI.
func extensionCompuesta(filename string) string {
	lower := strings.ToLower(filename)
	for _, suffix := range []string{".nii.gz", ".tar.gz"} {
		if strings.HasSuffix(lower, suffix) {
			return filename[len(filename)-len(suffix):]
		}
	}
	return filepath.Ext(filename)
}

// Create maneja POST /api/v1/consultas/:id/anexos (multipart, HU-07).
// El médico que atendió la consulta adjunta un resultado/imagen. El archivo se
// guarda en el volumen y sus metadatos en examinagen, ligado a la consulta.
func (h *AnexoHandler) Create(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	consultaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid consulta id"})
		return
	}

	fileHeader, err := c.FormFile("archivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "falta el archivo"})
		return
	}
	nombre := strings.TrimSpace(c.PostForm("nombre"))

	ctx := context.Background()

	// Médico autenticado.
	var medicoID uuid.UUID
	if err := h.pool.QueryRow(ctx, `SELECT id FROM medico WHERE usuario_id = $1`, userID).Scan(&medicoID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede adjuntar anexos"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	// La consulta debe existir y haber sido atendida por este médico.
	var pacienteID, historiaClinicaID, consultaMedicoID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT paciente_id, historia_clinica_id, medico_id FROM consulta WHERE id = $1`,
		consultaID,
	).Scan(&pacienteID, &historiaClinicaID, &consultaMedicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "la consulta no existe"})
			return
		}
		log.Printf("lookup consulta error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find consulta"})
		return
	}
	if consultaMedicoID != medicoID {
		c.JSON(http.StatusForbidden, gin.H{"error": "solo el médico que atendió la consulta puede adjuntar"})
		return
	}

	// Nombre de archivo en disco: <uuid><ext> (conserva la extensión original,
	// incluidas las compuestas como .nii.gz que MONAI necesita para reconocer
	// el formato NIfTI).
	ext := extensionCompuesta(fileHeader.Filename)
	storedName := uuid.NewString() + ext
	destPath := filepath.Join(h.uploadsDir, storedName)
	if err := c.SaveUploadedFile(fileHeader, destPath); err != nil {
		log.Printf("save file error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	if nombre == "" {
		// Por defecto, el nombre original sin extensión.
		nombre = strings.TrimSuffix(fileHeader.Filename, ext)
		if nombre == "" {
			nombre = "Anexo"
		}
	}
	categoria := categoriaPorExt(ext)

	var anexoID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`INSERT INTO examinagen
		   (historia_clinica_id, paciente_id, consulta_id, tipo_examen, descripcion,
		    url_imagen, medico_solicitante_id, fecha_examen, fecha_carga)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		 RETURNING id`,
		historiaClinicaID, pacienteID, consultaID, categoria, nombre, storedName, medicoID,
	).Scan(&anexoID)
	if err != nil {
		log.Printf("create anexo error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save anexo"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     anexoID,
		"nombre": nombre,
		"tipo":   categoria,
	})
}

// Serve maneja GET /api/v1/anexos/:id/archivo.
// Devuelve el archivo del anexo con su content-type (por extensión) para verlo.
func (h *AnexoHandler) Serve(c *gin.Context) {
	anexoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid anexo id"})
		return
	}

	var storedName, nombre string
	err = h.pool.QueryRow(context.Background(),
		`SELECT url_imagen, descripcion FROM examinagen WHERE id = $1`, anexoID,
	).Scan(&storedName, &nombre)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "anexo no encontrado"})
			return
		}
		log.Printf("lookup anexo error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch anexo"})
		return
	}

	// Evita path traversal: solo el nombre base.
	safe := filepath.Base(storedName)
	c.File(filepath.Join(h.uploadsDir, safe))
}
