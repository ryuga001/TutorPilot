// Package webhooks receives LiveKit's server callbacks.
//
// These are the events that cannot be observed from the request path: a recording
// finishes minutes after the lecture ends, and participants come and go without
// touching the API. Without them recordings never appear and attendance is empty.
package webhooks

import (
	"github.com/gin-gonic/gin"

	"tutorpilot/internal/modules/lecture"
)

type Module struct {
	handler *Handler
}

type Deps struct {
	Lectures *lecture.Service

	// LiveKit credentials, used to verify the signature on every callback.
	APIKey    string
	APISecret string
}

func New(d Deps) *Module {
	return &Module{handler: NewHandler(d)}
}

// RegisterRoutes mounts the callback endpoint. It sits outside RequireAuth — the
// caller is LiveKit, not a user — and authenticates by verifying the signed token
// LiveKit puts in the Authorization header against the shared API secret.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/livekit", m.handler.LiveKit)
}
