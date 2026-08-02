package batches

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	handler "tutorpilot/internal/modules/admin/handler/batches"
	repository "tutorpilot/internal/modules/admin/repository/batches"
	service "tutorpilot/internal/modules/admin/service/batches"
	"tutorpilot/internal/modules/admin/storage"
)

type Module struct {
	handler *handler.Handler
	repo    *repository.Repository
	deps    Deps
}

type Deps struct {
	DB      *pgxpool.Pool
	Storage *storage.Storage

	Pepper        string
	SignInURL     string
	Stream        string
	ImportMaxRows int
	RequireAuth   func() gin.HandlerFunc
	RequirePriv   func(privilege string) gin.HandlerFunc
}

func New(d Deps) *Module {
	repo := repository.NewRepository(d.DB)
	svc := service.NewService(repo, d.Storage, d.DB, service.ServiceConfig{
		Pepper:        d.Pepper,
		SignInURL:     d.SignInURL,
		Stream:        d.Stream,
		ImportMaxRows: d.ImportMaxRows,
	})
	return &Module{handler: handler.NewHandler(svc), repo: repo, deps: d}
}

func (m *Module) DriveWriter() *repository.Repository { return m.repo }

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
		g.POST("/:id/students/enroll", priv("batch.edit"), m.handler.EnrollStudents)
		g.POST("/:id/students/import", priv("batch.edit"), m.handler.ImportStudents)
		g.DELETE("/:id/students/:sid", priv("batch.edit"), m.handler.RemoveStudent)

		g.GET("/:id/drive", priv("batch.view"), m.handler.ListDrive)
		g.POST("/:id/drive/folders", priv("batch.edit"), m.handler.CreateFolder)
		g.POST("/:id/drive/files", priv("batch.edit"), m.handler.UploadFile)
		g.PUT("/:id/drive/:nid", priv("batch.edit"), m.handler.RenameNode)
		g.DELETE("/:id/drive/:nid", priv("batch.edit"), m.handler.DeleteNode)
	}
}
