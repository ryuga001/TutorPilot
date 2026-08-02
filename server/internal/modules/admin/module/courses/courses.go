package courses

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	handler "tutorpilot/internal/modules/admin/handler/courses"
	repository "tutorpilot/internal/modules/admin/repository/courses"
	service "tutorpilot/internal/modules/admin/service/courses"
	"tutorpilot/internal/modules/admin/storage"
)

type Module struct {
	handler *handler.Handler
	deps    Deps
}

type Deps struct {
	DB          *pgxpool.Pool
	Storage     *storage.Storage
	RequireAuth func() gin.HandlerFunc
	RequirePriv func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := repository.NewRepository(d.DB)
	svc := service.NewService(repo, d.Storage)
	return &Module{handler: handler.NewHandler(svc), deps: d}
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/courses")
	g.Use(m.deps.RequireAuth())

	priv := m.deps.RequirePriv
	{
		g.GET("", priv("course.view"), m.handler.List)
		g.POST("", priv("course.create"), m.handler.Create)
		g.GET("/:id", priv("course.view"), m.handler.Get)
		g.PUT("/:id", priv("course.edit"), m.handler.Update)
		g.DELETE("/:id", priv("course.delete"), m.handler.Delete)

		g.POST("/:id/publish", priv("course.edit"), m.handler.Publish)
		g.POST("/:id/unpublish", priv("course.edit"), m.handler.Unpublish)
		g.POST("/:id/thumbnail", priv("course.edit"), m.handler.UploadThumbnail)

		g.POST("/:id/modules", priv("course.edit"), m.handler.CreateModule)
		g.PUT("/:id/modules/:mid", priv("course.edit"), m.handler.UpdateModule)
		g.DELETE("/:id/modules/:mid", priv("course.edit"), m.handler.DeleteModule)

		g.POST("/:id/modules/:mid/lessons", priv("course.edit"), m.handler.CreateLesson)
		g.PUT("/:id/modules/:mid/lessons/:lid", priv("course.edit"), m.handler.UpdateLesson)
		g.DELETE("/:id/modules/:mid/lessons/:lid", priv("course.edit"), m.handler.DeleteLesson)

		g.GET("/:id/resources", priv("course.view"), m.handler.ListResources)
		g.POST("/:id/resources", priv("course.edit"), m.handler.UploadResource)
		g.DELETE("/:id/resources/:rid", priv("course.edit"), m.handler.DeleteResource)
	}
}
