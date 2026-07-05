package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/config"
	"sinapsis-backend/handlers"
	"sinapsis-backend/middleware"
)

type Handler struct {
	auth     *handlers.AuthHandler
	paciente *handlers.PacienteHandler
	consulta *handlers.ConsultaHandler
	cita     *handlers.CitaHandler
}

func Setup(r *gin.Engine, pool *pgxpool.Pool, cfg *config.Config) {
	h := &Handler{
		auth:     handlers.NewAuthHandler(pool, cfg),
		paciente: handlers.NewPacienteHandler(pool),
		consulta: handlers.NewConsultaHandler(pool),
		cita:     handlers.NewCitaHandler(pool),
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
			pacientes.GET("/:id", h.paciente.GetByID)
			pacientes.GET("/:id/consultas", h.consulta.ListByPaciente)
			pacientes.POST("", middleware.RequireAuth(cfg), h.paciente.Create)
			pacientes.POST("/:id/transferir", middleware.RequireAuth(cfg), h.paciente.Transfer)
		}

		api.GET("/medicos", middleware.RequireAuth(cfg), h.paciente.ListMedicos)

		consultas := api.Group("/consultas")
		{
			consultas.POST("", middleware.RequireAuth(cfg), h.consulta.Create)
		}

		citas := api.Group("/citas")
		{
			citas.POST("", middleware.RequireAuth(cfg), h.cita.Create)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.RequireAuth(cfg))
		{
			admin.GET("/usuarios", handlers.ObtenerUsuarios)
			admin.GET("/usuarios/:id", handlers.ObtenerUsuario)
			admin.POST("/usuarios", handlers.CrearUsuario)       // HU-19
			admin.PUT("/usuarios/:id", handlers.EditarUsuario)   // HU-20
			admin.DELETE("/usuarios/:id", handlers.EliminarUsuario) // HU-21
			admin.PATCH("/usuarios/:id/rol", handlers.AsignarRol) // HU-22 TO DO
		}
	}
}
