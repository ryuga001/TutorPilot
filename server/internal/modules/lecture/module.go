package lecture

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/livekit"
	"tutorpilot/internal/pkg/storage"
)

type Module struct {
	handler *Handler
	deps    Deps
}

type Deps struct {
	DB          *pgxpool.Pool
	LiveKit     *livekit.LiveKitClient
	Storage     *storage.Storage
	RequireAuth func() gin.HandlerFunc
	RequirePriv func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := NewRepository(d.DB)
	svc := NewService(repo, d.LiveKit, d.Storage)
	return &Module{handler: NewHandler(svc), deps: d}
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/lectures")
	g.Use(m.deps.RequireAuth())

	priv := m.deps.RequirePriv
	{
		g.POST("", priv("lecture.create"), m.handler.Create)
		g.GET("", priv("lecture.view"), m.handler.List)
		g.GET("/:id", priv("lecture.view"), m.handler.Get)
		g.PUT("/:id", priv("lecture.edit"), m.handler.Update)
		g.DELETE("/:id", priv("lecture.delete"), m.handler.Delete)

		g.POST("/:id/start", priv("lecture.control"), m.handler.Start)
		g.POST("/:id/end", priv("lecture.control"), m.handler.End)
		g.POST("/:id/join", priv("lecture.view"), m.handler.Join)
	}
}
