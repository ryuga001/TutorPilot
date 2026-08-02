package webhooks

import (
	"github.com/gin-gonic/gin"

	handler "tutorpilot/internal/modules/admin/handler/webhooks"
	lecture "tutorpilot/internal/modules/admin/service/lecture"
)

type Module struct {
	handler *handler.Handler
}

type Deps struct {
	Lectures *lecture.Service

	APIKey    string
	APISecret string
}

func New(d Deps) *Module {
	return &Module{handler: handler.NewHandler(handler.Deps{Lectures: d.Lectures, APIKey: d.APIKey, APISecret: d.APISecret})}
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/livekit", m.handler.LiveKit)
}
