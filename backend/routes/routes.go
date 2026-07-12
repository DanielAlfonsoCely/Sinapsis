package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/config"
	"sinapsis-backend/handlers"
	"sinapsis-backend/middleware"
)

type Handler struct {
	auth               *handlers.AuthHandler
	paciente           *handlers.PacienteHandler
	consulta           *handlers.ConsultaHandler
	cita               *handlers.CitaHandler
	entidad            *handlers.EntidadHandler
	formula            *handlers.FormulaHandler
	anexo              *handlers.AnexoHandler
	historiaClinicaPDF *handlers.HistoriaClinicaPDFHandler
	adminUsuario       *handlers.AdminUsuarioHandler
}

func Setup(r *gin.Engine, pool *pgxpool.Pool, cfg *config.Config) {
	h := &Handler{
		auth:               handlers.NewAuthHandler(pool, cfg),
		paciente:           handlers.NewPacienteHandler(pool),
		consulta:           handlers.NewConsultaHandler(pool),
		cita:               handlers.NewCitaHandler(pool),
		entidad:            handlers.NewEntidadHandler(pool),
		formula:            handlers.NewFormulaHandler(pool),
		anexo:              handlers.NewAnexoHandler(pool, cfg.UploadsDir),
		historiaClinicaPDF: handlers.NewHistoriaClinicaPDFHandler(pool),
		adminUsuario:       handlers.NewAdminUsuarioHandler(pool),
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
			pacientes.GET("", middleware.RequireAuth(cfg), h.paciente.List)
			pacientes.GET("/me", middleware.RequireAuth(cfg), h.paciente.Me)
			pacientes.GET("/:id", middleware.RequireAuth(cfg), h.paciente.GetByID)
			pacientes.GET("/:id/consultas", h.consulta.ListByPaciente)
			pacientes.GET("/:id/formulas", h.formula.ListByPaciente)
			pacientes.GET("/:id/historia-clinica/pdf", middleware.RequireAuth(cfg), h.historiaClinicaPDF.ExportPDF)
			pacientes.POST("", middleware.RequireAuth(cfg), h.paciente.Create)
			pacientes.POST("/:id/remisiones", middleware.RequireAuth(cfg), h.paciente.AutorizarEspecialidad)
		}

		api.GET("/especialidades", middleware.RequireAuth(cfg), h.paciente.ListEspecialidades)
		api.GET("/mi/agenda", middleware.RequireAuth(cfg), h.paciente.MiAgenda)

		consultas := api.Group("/consultas")
		{
			consultas.POST("", middleware.RequireAuth(cfg), h.consulta.Create)
			consultas.POST("/:id/anexos", middleware.RequireAuth(cfg), h.anexo.Create)
		}

		api.GET("/anexos/:id/archivo", middleware.RequireAuth(cfg), h.anexo.Serve)

		citas := api.Group("/citas")
		{
			citas.GET("/hoy", middleware.RequireAuth(cfg), h.cita.CitasHoy)
			citas.GET("/semana", middleware.RequireAuth(cfg), h.cita.CitasSemana)
			citas.POST("", middleware.RequireAuth(cfg), h.cita.Create)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.RequireAuth(cfg), middleware.RequireAdmin(cfg))
		{
			admin.GET("/usuarios", h.adminUsuario.ListUsuarios)
			admin.PATCH("/usuarios/:id/rol", h.adminUsuario.PatchRol)
			admin.GET("/entidades", h.entidad.ListAdmin)
			admin.GET("/entidades/:id", h.entidad.GetByIDAdmin)
			admin.GET("/stats", h.entidad.Stats)
		}

		entidades := api.Group("/entidades")
		{
			entidades.GET("", middleware.RequireAuth(cfg), h.entidad.List)
			entidades.POST("", middleware.RequireAuth(cfg), h.entidad.Create)
		}

		formulas := api.Group("/formulas")
		{
			formulas.POST("", middleware.RequireAuth(cfg), h.formula.Create)
			formulas.POST("/:id/anular", middleware.RequireAuth(cfg), h.formula.Anular)
		}
	}
}
