package tutors

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/pkg/mailer"
	"tutorpilot/internal/pkg/storage"
)

type Module struct {
	handler *Handler
	deps    Deps
}

type Deps struct {
	DB          *pgxpool.Pool
	Storage     *storage.Storage
	Mailer      *mailer.Mailer
	Pepper      string
	RequireAuth func() gin.HandlerFunc
	RequirePriv func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := NewRepository(d.DB)
	svc := NewService(repo, d.Storage, d.DB, d.Mailer, d.Pepper)
	return &Module{handler: NewHandler(svc), deps: d}
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
