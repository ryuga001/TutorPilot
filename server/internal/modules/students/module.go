package students

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/pkg/storage"
)

type Module struct {
	handler *Handler
	deps    Deps
}

type Deps struct {
	DB          *pgxpool.Pool
	Storage     *storage.Storage
	RequireAuth func() gin.HandlerFunc
	RequirePriv func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := NewRepository(d.DB)
	svc := NewService(repo, d.Storage, d.DB)
	return &Module{handler: NewHandler(svc), deps: d}
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/students")
	g.Use(m.deps.RequireAuth())

	priv := m.deps.RequirePriv
	{
		g.GET("", priv("student.view"), m.handler.List)
		g.POST("", priv("student.create"), m.handler.Create)
		g.GET("/:id", priv("student.view"), m.handler.Get)
		g.PUT("/:id", priv("student.edit"), m.handler.Update)
		g.DELETE("/:id", priv("student.delete"), m.handler.Delete)
		g.POST("/:id/profile-image", priv("student.edit"), m.handler.UploadProfileImage)
	}
}
