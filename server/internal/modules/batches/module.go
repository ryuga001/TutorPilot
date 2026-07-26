package batches

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/pkg/storage"
)

type Module struct {
	handler *Handler
	deps    Deps
}

type Deps struct {
	DB          *pgxpool.Pool
	Storage     *storage.Storage
	Notifier    *notification.Notifier
	RequireAuth func() gin.HandlerFunc
	RequirePriv func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := NewRepository(d.DB)
	svc := NewService(repo, d.Storage, d.Notifier)
	return &Module{handler: NewHandler(svc), deps: d}
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/batches")
	g.Use(m.deps.RequireAuth())

	priv := m.deps.RequirePriv
	{
		g.GET("", priv("batch.view"), m.handler.List)
		g.POST("", priv("batch.create"), m.handler.Create)
		g.GET("/:id", priv("batch.view"), m.handler.Get)
		g.PUT("/:id", priv("batch.edit"), m.handler.Update)
		g.DELETE("/:id", priv("batch.delete"), m.handler.Delete)

		g.POST("/:id/publish", priv("batch.edit"), m.handler.Publish)
		g.POST("/:id/unpublish", priv("batch.edit"), m.handler.Unpublish)

		g.PUT("/:id/modules/:mid/assign", priv("batch.edit"), m.handler.AssignTutor)
		g.DELETE("/:id/modules/:mid/assign", priv("batch.edit"), m.handler.UnassignTutor)

		g.GET("/:id/tutors", priv("batch.view"), m.handler.ListTutors)

		g.GET("/:id/students", priv("batch.view"), m.handler.ListStudents)
		g.POST("/:id/students/import", priv("batch.edit"), m.handler.ImportStudents)
		g.DELETE("/:id/students/:sid", priv("batch.edit"), m.handler.RemoveStudent)

		g.GET("/:id/drive", priv("batch.view"), m.handler.ListDrive)
		g.POST("/:id/drive/folders", priv("batch.edit"), m.handler.CreateFolder)
		g.POST("/:id/drive/files", priv("batch.edit"), m.handler.UploadFile)
		g.PUT("/:id/drive/:nid", priv("batch.edit"), m.handler.RenameNode)
		g.DELETE("/:id/drive/:nid", priv("batch.edit"), m.handler.DeleteNode)
	}
}
