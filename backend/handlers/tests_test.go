package handlers

// =============================================================================
// TESTS UNITARIOS — HU 02, 03, 04, 06 y 07
// =============================================================================
//
// Estos son tests UNITARIOS: NO tocan la base de datos. Verifican:
//   1. Funciones puras (categorizar anexos, generar contraseña, parsear fechas).
//   2. El middleware de autenticación JWT (la puerta de todas estas HU).
//   3. La validación de entrada de cada handler (los caminos que responden
//      ANTES de consultar la base de datos: 400/401 por datos inválidos).
//
// Las reglas de negocio que dependen de datos (ej: "solo el médico que atendió
// puede adjuntar", "el paciente debe tener cita activa hoy", rollback de la
// transacción) se probarán con tests de INTEGRACIÓN contra una base real.
//
// Todo en un solo archivo a propósito, para que sea fácil de revisar.
// Ejecutar con:  go test ./handlers/ -v
// =============================================================================

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"sinapsis-backend/config"
	"sinapsis-backend/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

// un UUID válido cualquiera, para pasar validaciones de formato.
const someUUID = "11111111-1111-1111-1111-111111111111"

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// newCtx arma un *gin.Context de prueba. Si setUser es true, simula que el
// middleware ya autenticó y dejó user_id en el contexto.
func newCtx(method string, setUser bool, userID string, body io.Reader, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, "/", body)
	c.Params = params
	if setUser {
		c.Set("user_id", userID)
	}
	return c, rec
}

// postJSON arma un POST con cuerpo JSON y (opcionalmente) user_id en contexto.
func postJSON(setUser bool, userID, jsonBody string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	c, rec := newCtx(http.MethodPost, setUser, userID, strings.NewReader(jsonBody), params)
	c.Request.Header.Set("Content-Type", "application/json")
	return c, rec
}

func errMsg(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		return ""
	}
	if e, ok := body["error"].(string); ok {
		return e
	}
	return ""
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d; se esperaba %d (body: %s)", rec.Code, want, rec.Body.String())
	}
}

func makeToken(secret string, claims jwt.MapClaims) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString([]byte(secret))
	return s
}

// =============================================================================
// HU-07 — categoriaPorExt (función pura)
// =============================================================================

func TestCategoriaPorExt(t *testing.T) {
	casos := []struct {
		ext  string
		want string
	}{
		{".jpg", "imagen"},
		{".jpeg", "imagen"},
		{".png", "imagen"},
		{".dcm", "imagen"},   // DICOM médico
		{".JPG", "imagen"},   // mayúsculas -> normaliza
		{".PnG", "imagen"},   // mixto
		{".pdf", "documento"},
		{".txt", "documento"},
		{".docx", "documento"},
		{"", "documento"},         // archivo sin extensión
		{".exe", "documento"},     // extensión desconocida
		{"jpg", "documento"},      // sin el punto -> no matchea el switch
	}
	for _, tc := range casos {
		if got := categoriaPorExt(tc.ext); got != tc.want {
			t.Errorf("categoriaPorExt(%q) = %q; se esperaba %q", tc.ext, got, tc.want)
		}
	}
}

// =============================================================================
// HU-02 — generateTempPassword (función pura)
// =============================================================================

func TestGenerateTempPasswordLongitud(t *testing.T) {
	for _, n := range []int{1, 8, 12, 32} {
		p, err := generateTempPassword(n)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(p) != n {
			t.Errorf("longitud = %d; se esperaba %d", len(p), n)
		}
	}
}

func TestGenerateTempPasswordLongitudCero(t *testing.T) {
	p, err := generateTempPassword(0)
	if err != nil || p != "" {
		t.Errorf("con 0 se esperaba (\"\", nil); se obtuvo (%q, %v)", p, err)
	}
}

func TestGenerateTempPasswordSoloUsaCharsetPermitido(t *testing.T) {
	p, err := generateTempPassword(500)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	for _, r := range p {
		if !strings.ContainsRune(tempPasswordChars, r) {
			t.Fatalf("la contraseña contiene un carácter fuera del charset: %q", r)
		}
	}
}

func TestGenerateTempPasswordSinCaracteresAmbiguos(t *testing.T) {
	// El charset excluye a propósito caracteres confundibles: 0 O 1 I l.
	for _, amb := range []rune{'0', 'O', '1', 'I', 'l'} {
		if strings.ContainsRune(tempPasswordChars, amb) {
			t.Errorf("el charset no debería incluir el carácter ambiguo %q", amb)
		}
	}
}

func TestGenerateTempPasswordEsAleatoria(t *testing.T) {
	a, _ := generateTempPassword(16)
	b, _ := generateTempPassword(16)
	if a == b {
		t.Errorf("dos contraseñas seguidas fueron idénticas (%q); debería ser aleatorio", a)
	}
}

// =============================================================================
// Middleware de autenticación (puerta de HU 02-07)
// =============================================================================

func runAuth(header string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if header != "" {
		c.Request.Header.Set("Authorization", header)
	}
	middleware.RequireAuth(&config.Config{JWTSecret: "test-secret"})(c)
	return c, rec
}

func TestRequireAuthSinHeader(t *testing.T) {
	_, rec := runAuth("")
	wantStatus(t, rec, http.StatusUnauthorized)
}

func TestRequireAuthEsquemaInvalido(t *testing.T) {
	// Sin el prefijo "Bearer " debe rechazar.
	_, rec := runAuth("Token abc.def.ghi")
	wantStatus(t, rec, http.StatusUnauthorized)
}

func TestRequireAuthTokenBasura(t *testing.T) {
	_, rec := runAuth("Bearer no-es-un-jwt")
	wantStatus(t, rec, http.StatusUnauthorized)
}

func TestRequireAuthFirmaConSecretoEquivocado(t *testing.T) {
	tok := makeToken("otro-secreto", jwt.MapClaims{
		"user_id": someUUID,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	_, rec := runAuth("Bearer " + tok)
	wantStatus(t, rec, http.StatusUnauthorized)
}

func TestRequireAuthTokenExpirado(t *testing.T) {
	tok := makeToken("test-secret", jwt.MapClaims{
		"user_id": someUUID,
		"exp":     time.Now().Add(-time.Hour).Unix(), // expiró hace 1h
	})
	_, rec := runAuth("Bearer " + tok)
	wantStatus(t, rec, http.StatusUnauthorized)
}

func TestRequireAuthSinUserID(t *testing.T) {
	// Token válido pero sin claim user_id -> debe rechazar.
	tok := makeToken("test-secret", jwt.MapClaims{
		"tipo_usuario": "medico",
		"exp":          time.Now().Add(time.Hour).Unix(),
	})
	_, rec := runAuth("Bearer " + tok)
	wantStatus(t, rec, http.StatusUnauthorized)
}

func TestRequireAuthTokenValido(t *testing.T) {
	tok := makeToken("test-secret", jwt.MapClaims{
		"user_id":      someUUID,
		"tipo_usuario": "medico",
		"exp":          time.Now().Add(time.Hour).Unix(),
	})
	c, rec := runAuth("Bearer " + tok)
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("un token válido no debería ser rechazado (body: %s)", rec.Body.String())
	}
	if uid, ok := c.Get("user_id"); !ok || uid != someUUID {
		t.Errorf("no se expuso user_id en el contexto: got=%v ok=%v", uid, ok)
	}
	if tu, _ := c.Get("tipo_usuario"); tu != "medico" {
		t.Errorf("no se expuso tipo_usuario correctamente: %v", tu)
	}
}

// =============================================================================
// Autenticación — Iniciar sesión (AuthHandler.Login)
// =============================================================================
//
// Login no depende de sesión previa (es el propio endpoint que la crea), así
// que el único camino que no toca la BD es el de validación del body. Los
// casos de éxito (credenciales correctas, emisión de JWT) y de error que sí
// dependen de datos (email no registrado, contraseña incorrecta) requieren
// una base real y quedan para tests de integración, igual que el resto de
// reglas de negocio de este archivo.

func TestAuthLoginValidacion(t *testing.T) {
	h := NewAuthHandler(nil, &config.Config{JWTSecret: "test-secret"})

	t.Run("json inválido -> 400", func(t *testing.T) {
		c, rec := postJSON(false, "", `{ roto`, nil)
		h.Login(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("falta email (required) -> 400", func(t *testing.T) {
		c, rec := postJSON(false, "", `{"contrasena":"secreta123"}`, nil)
		h.Login(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("email inválido -> 400", func(t *testing.T) {
		c, rec := postJSON(false, "", `{"email":"no-es-correo","contrasena":"secreta123"}`, nil)
		h.Login(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("falta contrasena (required) -> 400", func(t *testing.T) {
		c, rec := postJSON(false, "", `{"email":"ana@mail.com"}`, nil)
		h.Login(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
}

// =============================================================================
// HU-02 — Registrar paciente: validación de entrada (PacienteHandler.Create)
// =============================================================================

func TestPacienteCreateValidacion(t *testing.T) {
	h := NewPacienteHandler(nil) // no se llega a la BD en estos caminos

	// cuerpo base válido; cada caso lo altera.
	base := `{"numero_documento":"123","tipo_documento":"CC","nombre_paciente":"Ana","apellidos_paciente":"Ruiz","fecha_nacimiento":"1990-01-01","email":"ana@mail.com"}`

	t.Run("sin sesión -> 401", func(t *testing.T) {
		c, rec := postJSON(false, "", base, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
	t.Run("user_id no es uuid -> 401", func(t *testing.T) {
		c, rec := postJSON(true, "no-uuid", base, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
	t.Run("json inválido -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{ roto`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("falta email (required) -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{"numero_documento":"1","tipo_documento":"CC","nombre_paciente":"A","apellidos_paciente":"B","fecha_nacimiento":"1990-01-01"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("email inválido -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{"numero_documento":"1","tipo_documento":"CC","nombre_paciente":"A","apellidos_paciente":"B","fecha_nacimiento":"1990-01-01","email":"no-es-correo"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("tipo_documento fuera del enum -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{"numero_documento":"1","tipo_documento":"XX","nombre_paciente":"A","apellidos_paciente":"B","fecha_nacimiento":"1990-01-01","email":"a@b.com"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("sexo fuera del enum -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{"numero_documento":"1","tipo_documento":"CC","nombre_paciente":"A","apellidos_paciente":"B","fecha_nacimiento":"1990-01-01","email":"a@b.com","sexo":"Z"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("fecha_nacimiento con formato inválido -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{"numero_documento":"1","tipo_documento":"CC","nombre_paciente":"A","apellidos_paciente":"B","fecha_nacimiento":"01/01/1990","email":"a@b.com"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
		if msg := errMsg(t, rec); !strings.Contains(msg, "fecha_nacimiento") {
			t.Errorf("el error debería mencionar fecha_nacimiento; fue: %q", msg)
		}
	})
}

func TestPacienteListRequiereMedico(t *testing.T) {
	h := NewPacienteHandler(nil)
	t.Run("sin sesión -> 401", func(t *testing.T) {
		c, rec := newCtx(http.MethodGet, false, "", nil, nil)
		h.List(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
	t.Run("user_id inválido -> 401", func(t *testing.T) {
		c, rec := newCtx(http.MethodGet, true, "no-uuid", nil, nil)
		h.List(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
}

// =============================================================================
// HU-03 — Registrar consulta: validación de entrada (ConsultaHandler.Create)
// =============================================================================

func TestConsultaCreateValidacion(t *testing.T) {
	h := NewConsultaHandler(nil)

	t.Run("sin sesión -> 401", func(t *testing.T) {
		c, rec := postJSON(false, "", `{"paciente_id":"`+someUUID+`","motivo_consulta":"dolor"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
	t.Run("user_id inválido -> 401", func(t *testing.T) {
		c, rec := postJSON(true, "no-uuid", `{"paciente_id":"`+someUUID+`","motivo_consulta":"dolor"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
	t.Run("json inválido -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("falta motivo_consulta (required) -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{"paciente_id":"`+someUUID+`"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("paciente_id no es uuid -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{"paciente_id":"abc","motivo_consulta":"dolor"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("proxima_cita con formato inválido -> 400", func(t *testing.T) {
		c, rec := postJSON(true, someUUID, `{"paciente_id":"`+someUUID+`","motivo_consulta":"dolor","proxima_cita":"15-07-2026"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
		if msg := errMsg(t, rec); !strings.Contains(msg, "proxima_cita") {
			t.Errorf("el error debería mencionar proxima_cita; fue: %q", msg)
		}
	})
}

// =============================================================================
// HU-04 — Ver historia clínica: validación (ConsultaHandler.ListByPaciente)
// =============================================================================

// NOTA: antes esta ruta no exigía sesión (era el bug de ruta sin RequireAuth
// que se corrigió en routes.go). Ahora ListByPaciente necesita el user_id
// para auditar quién consultó, así que el test simula una sesión válida y
// ejercita específicamente la validación del id de paciente.
func TestConsultaListByPacienteValidacion(t *testing.T) {
	h := NewConsultaHandler(nil)
	t.Run("id de paciente inválido -> 400", func(t *testing.T) {
		c, rec := newCtx(http.MethodGet, true, someUUID, nil, gin.Params{{Key: "id", Value: "no-uuid"}})
		h.ListByPaciente(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("sin sesión -> 401", func(t *testing.T) {
		c, rec := newCtx(http.MethodGet, false, "", nil, gin.Params{{Key: "id", Value: someUUID}})
		h.ListByPaciente(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
}

// =============================================================================
// HU-06 — Fórmulas médicas: validación (FormulaHandler)
// =============================================================================

func TestFormulaCreateAuth(t *testing.T) {
	h := NewFormulaHandler(nil)
	// resolveMedico se ejecuta primero; sin sesión válida corta antes de la BD.
	t.Run("sin sesión -> 401", func(t *testing.T) {
		c, rec := postJSON(false, "", `{}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
	t.Run("user_id inválido -> 401", func(t *testing.T) {
		c, rec := postJSON(true, "no-uuid", `{}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
}

// NOTA: mismo cambio de contrato que en Consulta -- ListByPaciente ahora
// exige sesión para poder auditar.
func TestFormulaListByPacienteValidacion(t *testing.T) {
	h := NewFormulaHandler(nil)
	t.Run("id de paciente inválido -> 400", func(t *testing.T) {
		c, rec := newCtx(http.MethodGet, true, someUUID, nil, gin.Params{{Key: "id", Value: "xx"}})
		h.ListByPaciente(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("sin sesión -> 401", func(t *testing.T) {
		c, rec := newCtx(http.MethodGet, false, "", nil, gin.Params{{Key: "id", Value: someUUID}})
		h.ListByPaciente(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
}

func TestFormulaAnularAuth(t *testing.T) {
	h := NewFormulaHandler(nil)
	t.Run("sin sesión -> 401", func(t *testing.T) {
		c, rec := newCtx(http.MethodPost, false, "", nil, gin.Params{{Key: "id", Value: someUUID}})
		h.Anular(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
}

// =============================================================================
// HU-07 — Anexos (AnexoHandler): validación de entrada
// =============================================================================

func TestAnexoCreateValidacion(t *testing.T) {
	h := NewAnexoHandler(nil, "/tmp")

	t.Run("sin sesión -> 401", func(t *testing.T) {
		c, rec := newCtx(http.MethodPost, false, "", nil, gin.Params{{Key: "id", Value: someUUID}})
		h.Create(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
	t.Run("user_id inválido -> 401", func(t *testing.T) {
		c, rec := newCtx(http.MethodPost, true, "no-uuid", nil, gin.Params{{Key: "id", Value: someUUID}})
		h.Create(c)
		wantStatus(t, rec, http.StatusUnauthorized)
	})
	t.Run("consulta id inválido -> 400", func(t *testing.T) {
		c, rec := newCtx(http.MethodPost, true, someUUID, nil, gin.Params{{Key: "id", Value: "no-uuid"}})
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("sin archivo adjunto -> 400", func(t *testing.T) {
		c, rec := newCtx(http.MethodPost, true, someUUID, nil, gin.Params{{Key: "id", Value: someUUID}})
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
		if msg := errMsg(t, rec); !strings.Contains(msg, "archivo") {
			t.Errorf("el error debería mencionar el archivo faltante; fue: %q", msg)
		}
	})
}

func TestAnexoServeValidacion(t *testing.T) {
	h := NewAnexoHandler(nil, "/tmp")
	t.Run("anexo id inválido -> 400", func(t *testing.T) {
		c, rec := newCtx(http.MethodGet, false, "", nil, gin.Params{{Key: "id", Value: "no-uuid"}})
		h.Serve(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
}

// =============================================================================
// Entidades — Registrar entidad (EntidadHandler.Create)
// =============================================================================
//
// Al igual que Login, Create no requiere sesión previa a nivel de handler (el
// middleware de admin se aplica en las rutas, no aquí), así que el único
// camino sin BD es la validación de entrada. El caso de éxito (INSERT
// exitoso) y el de NIT duplicado (23505 -> 409) dependen de datos reales y
// quedan para tests de integración.

func TestEntidadCreateValidacion(t *testing.T) {
	h := NewEntidadHandler(nil)

	t.Run("json inválido -> 400", func(t *testing.T) {
		c, rec := postJSON(false, "", `{ roto`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("falta nombre_entidad (required) -> 400", func(t *testing.T) {
		c, rec := postJSON(false, "", `{"tipo_entidad":"IPS","nit":"900123456-1"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("tipo_entidad fuera del enum -> 400", func(t *testing.T) {
		c, rec := postJSON(false, "", `{"nombre_entidad":"Clínica Sur","tipo_entidad":"clinica_privada","nit":"900123456-1"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("falta nit (required) -> 400", func(t *testing.T) {
		c, rec := postJSON(false, "", `{"nombre_entidad":"Clínica Sur","tipo_entidad":"IPS"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("nombre_entidad excede longitud máxima -> 400", func(t *testing.T) {
		largo := strings.Repeat("a", 151)
		c, rec := postJSON(false, "", `{"nombre_entidad":"`+largo+`","tipo_entidad":"IPS","nit":"900123456-1"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("nit excede longitud máxima -> 400", func(t *testing.T) {
		largo := strings.Repeat("1", 51)
		c, rec := postJSON(false, "", `{"nombre_entidad":"Clínica Sur","tipo_entidad":"IPS","nit":"`+largo+`"}`, nil)
		h.Create(c)
		wantStatus(t, rec, http.StatusBadRequest)
	})
}

// asegura que uuid siga importado aunque se refactoricen casos.
var _ = uuid.Nil
