package tutors

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	handler "tutorpilot/internal/modules/admin/handler/tutors"
	repository "tutorpilot/internal/modules/admin/repository/tutors"
	service "tutorpilot/internal/modules/admin/service/tutors"
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
	g := rg.Group("/tutors")
	g.Use(m.deps.RequireAuth())

	priv := m.deps.RequirePriv
	{
		g.GET("", priv("tutor.view"), m.handler.List)
		g.POST("", priv("tutor.create"), m.handler.Create)
		g.GET("/:id", priv("tutor.view"), m.handler.Get)
		g.PUT("/:id", priv("tutor.edit"), m.handler.Update)
		g.DELETE("/:id", priv("tutor.delete"), m.handler.Delete)
		g.POST("/:id/profile-image", priv("tutor.edit"), m.handler.UploadProfileImage)
	}
}
