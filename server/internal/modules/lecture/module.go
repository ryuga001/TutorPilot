package lecture

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/livekit"
	"tutorpilot/internal/pkg/storage"
)

type Module struct {
	handler *Handler
	svc     *Service
	deps    Deps
}

type Deps struct {
	DB      *pgxpool.Pool
	LiveKit *livekit.LiveKitClient
	Storage *storage.Storage

	// HasPrivilege decides publish rights at join time, so grants follow the same
	// role definitions as every other authorization decision.
	HasPrivilege PrivilegeChecker
	JoinTokenTTL time.Duration

	RequireAuth func() gin.HandlerFunc
	RequirePriv func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := NewRepository(d.DB)
	svc := NewService(repo, d.LiveKit, d.Storage, d.HasPrivilege, d.JoinTokenTTL)
	return &Module{handler: NewHandler(svc), svc: svc, deps: d}
}

// Service exposes the service so the webhook module can hand LiveKit events to it.
func (m *Module) Service() *Service { return m.svc }

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
		g.POST("/:id/cancel", priv("lecture.control"), m.handler.Cancel)

		// Joining is its own privilege: students hold lecture.join but not
		// lecture.control, and publish rights are decided inside the handler.
		g.POST("/:id/join", priv("lecture.join"), m.handler.Join)

		g.GET("/:id/attendance", priv("lecture.view"), m.handler.Attendance)
		g.GET("/:id/recording", priv("recording.view"), m.handler.Recording)
	}
}
