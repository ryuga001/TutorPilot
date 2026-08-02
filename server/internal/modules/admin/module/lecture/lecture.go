package lecture

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	handler "tutorpilot/internal/modules/admin/handler/lecture"
	"tutorpilot/internal/modules/admin/livekit"
	repository "tutorpilot/internal/modules/admin/repository/lecture"
	service "tutorpilot/internal/modules/admin/service/lecture"
	"tutorpilot/internal/modules/admin/storage"
)

type Module struct {
	handler *handler.Handler
	svc     *service.Service
	deps    Deps
}

type Deps struct {
	DB      *pgxpool.Pool
	LiveKit *livekit.LiveKitClient
	Storage *storage.Storage

	HasPrivilege service.PrivilegeChecker
	JoinTokenTTL time.Duration

	RequireAuth func() gin.HandlerFunc
	RequirePriv func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := repository.NewRepository(d.DB)
	svc := service.NewService(repo, d.LiveKit, d.Storage, d.HasPrivilege, d.JoinTokenTTL)
	return &Module{handler: handler.NewHandler(svc), svc: svc, deps: d}
}

func (m *Module) Service() *service.Service { return m.svc }

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

		g.POST("/:id/join", priv("lecture.join"), m.handler.Join)

		g.GET("/:id/attendance", priv("lecture.view"), m.handler.Attendance)
		g.GET("/:id/recording", priv("recording.view"), m.handler.Recording)
	}
}
