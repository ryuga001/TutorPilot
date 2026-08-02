package students

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	handler "tutorpilot/internal/modules/admin/handler/students"
	repository "tutorpilot/internal/modules/admin/repository/students"
	service "tutorpilot/internal/modules/admin/service/students"
	"tutorpilot/internal/modules/admin/storage"
)

type Module struct {
	handler *handler.Handler
	deps    Deps
}

type Deps struct {
	DB          *pgxpool.Pool
	Storage     *storage.Storage
	Pepper      string
	SignInURL   string
	Stream      string
	RequireAuth func() gin.HandlerFunc
	RequirePriv func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := repository.NewRepository(d.DB)
	svc := service.NewService(repo, d.Storage, d.DB, d.Pepper, d.SignInURL, d.Stream)
	return &Module{handler: handler.NewHandler(svc), deps: d}
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
