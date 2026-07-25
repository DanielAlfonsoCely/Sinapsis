package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/audit"
	"sinapsis-backend/config"
	"sinapsis-backend/db/repositories"
	"sinapsis-backend/handlers"
	"sinapsis-backend/middleware"
	"sinapsis-backend/queue"
	"sinapsis-backend/services"
)

type Handler struct {
	auth               *handlers.AuthHandler
	usuario            *handlers.UsuarioHandler
	paciente           *handlers.PacienteHandler
	consulta           *handlers.ConsultaHandler
	cita               *handlers.CitaHandler
	entidad            *handlers.EntidadHandler
	formula            *handlers.FormulaHandler
	anexo              *handlers.AnexoHandler
	auditoria          *handlers.AuditoriaHandler
	historiaClinicaPDF *handlers.HistoriaClinicaPDFHandler
	adminUsuario       *handlers.AdminUsuarioHandler
	analisisIA         *handlers.AnalisisIAHandler
}

func Setup(r *gin.Engine, pool *pgxpool.Pool, cfg *config.Config, publisher *queue.Publisher) {
	// --- Auditoría: repo + observer + publisher, se arman una sola vez ---
	auditRepo := repositories.NewAuditRepository(pool)
	dbObserver := audit.NewDBAuditObserver(auditRepo)
	auditPublisher := audit.NewPublisher(dbObserver)

	auditService := services.NewAuditService(auditRepo)

	// --- Usuarios ---
	usuarioRepo := repositories.NewUsuarioRepository(pool)
	usuarioService := services.NewUsuarioService(usuarioRepo, auditPublisher)

	// --- Consulta / Cita / Fórmula: ahora con capa de servicio + auditoría,
	// siguiendo el mismo patrón (Publisher/Observer) que ya usa Usuario ---
	consultaRepo := repositories.NewConsultaRepository(pool)
	consultaService := services.NewConsultaService(consultaRepo, auditPublisher)

	citaRepo := repositories.NewCitaRepository(pool)
	citaService := services.NewCitaService(citaRepo, auditPublisher)

	formulaRepo := repositories.NewFormulaRepository(pool)
	formulaService := services.NewFormulaService(formulaRepo, auditPublisher)

	h := &Handler{
		auth:               handlers.NewAuthHandler(pool, cfg),
		usuario:            handlers.NewUsuarioHandler(usuarioService),
		paciente:           handlers.NewPacienteHandler(pool),
		consulta:           handlers.NewConsultaHandler(consultaService),
		cita:               handlers.NewCitaHandler(citaService),
		entidad:            handlers.NewEntidadHandler(pool),
		formula:            handlers.NewFormulaHandler(formulaService),
		anexo:              handlers.NewAnexoHandler(pool, cfg.UploadsDir),
		auditoria:          handlers.NewAuditoriaHandler(auditService),
		historiaClinicaPDF: handlers.NewHistoriaClinicaPDFHandler(pool),
		adminUsuario:       handlers.NewAdminUsuarioHandler(pool),
		analisisIA:         handlers.NewAnalisisIAHandler(pool, publisher),
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.auth.Register)
			auth.POST("/login", h.auth.Login)
		}

		pacientes := api.Group("/pacientes")
		{
			pacientes.GET("", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.paciente.List)
			pacientes.GET("/me", middleware.RequireAuth(cfg), middleware.RequireRole("paciente"), h.paciente.Me)
			pacientes.GET("/:id", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.paciente.GetByID)
			// FIX: estas dos rutas no tenían RequireAuth. Ahora lo necesitan
			// además por auditoría: ConsultaService.ListByPaciente y
			// FormulaService.ListByPaciente registran quién consultó el
			// historial/fórmulas de este paciente (tipo_operacion='consultar'),
			// y para eso necesitan el user_id que RequireAuth deja en el contexto.
			pacientes.GET("/:id/consultas", middleware.RequireAuth(cfg), h.consulta.ListByPaciente)
			pacientes.GET("/:id/formulas", middleware.RequireAuth(cfg), h.formula.ListByPaciente)
			pacientes.POST("", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.paciente.Create)
			pacientes.POST("/:id/remisiones", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.paciente.AutorizarEspecialidad)
			pacientes.GET("/:id/historia-clinica/pdf", middleware.RequireAuth(cfg), h.historiaClinicaPDF.ExportPDF)
		}

		api.GET("/especialidades", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.paciente.ListEspecialidades)
		api.GET("/mi/agenda", middleware.RequireAuth(cfg), middleware.RequireRole("paciente"), h.paciente.MiAgenda)

		consultas := api.Group("/consultas")
		{
			consultas.POST("", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.consulta.Create)
			consultas.POST("/:id/anexos", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.anexo.Create)
			consultas.PATCH("/:id/pre-diagnostico", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.consulta.UpdatePreDiagnostico)
		}

		api.GET("/anexos/:id/archivo", middleware.RequireAuth(cfg), h.anexo.Serve)

		citas := api.Group("/citas")
		{
			citas.GET("/hoy", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.cita.CitasHoy)
			citas.GET("/semana", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.cita.CitasSemana)
			citas.GET("/disponibilidad", middleware.RequireAuth(cfg), h.cita.Disponibilidad)
			citas.POST("", middleware.RequireAuth(cfg), middleware.RequireRole("paciente"), h.cita.Create)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.RequireAuth(cfg), middleware.RequireRole("admin_plataforma"))
		{
			admin.GET("/usuarios", h.adminUsuario.ListUsuarios)
			admin.POST("/usuarios", h.usuario.CrearUsuario)
			admin.PUT("/usuarios/:id", h.usuario.EditarUsuario)
			admin.DELETE("/usuarios/:id", h.usuario.EliminarUsuario)
			admin.PATCH("/usuarios/:id/rol", h.usuario.AsignarRol)
			admin.GET("/auditoria", h.auditoria.List)
			admin.GET("/entidades", h.entidad.ListAdmin)
			admin.GET("/entidades/:id", h.entidad.GetByIDAdmin)
			admin.GET("/entidades/:id/pacientes", h.entidad.ListPacientesAdmin)
			admin.PUT("/entidades/:id", h.entidad.UpdateAdmin)
			admin.GET("/stats", h.entidad.Stats)
		}

		entidades := api.Group("/entidades")
		{
			entidades.GET("", middleware.RequireAuth(cfg), h.entidad.List)
			entidades.POST("", middleware.RequireAuth(cfg), h.entidad.Create)
		}

		formulas := api.Group("/formulas")
		{
			formulas.POST("", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.formula.Create)
			formulas.POST("/:id/anular", middleware.RequireAuth(cfg), middleware.RequireRole("medico"), h.formula.Anular)
		}

		// Módulo de análisis IA (MONAI) — conectado al microservicio vía RabbitMQ
		examenes := api.Group("/examenes")
		{
			examenes.POST("/:id/analisis-ia", middleware.RequireAuth(cfg), h.analisisIA.SolicitarAnalisis)
		}

		sugerencias := api.Group("/sugerencias-ia")
		{
			sugerencias.GET("/:id", middleware.RequireAuth(cfg), h.analisisIA.GetSugerencia)
			sugerencias.PATCH("/:id/revision", middleware.RequireAuth(cfg), h.analisisIA.Revision)
		}
	}
}
