package auth

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"tutorpilot/internal/middleware"
	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/pkg/jwtutil"
)

type Module struct {
	handler *Handler
	svc     *Service
	jwt     *jwtutil.Manager
}

type Deps struct {
	DB         *pgxpool.Pool
	Redis      *redis.Client
	JWT        *jwtutil.Manager
	Notifier   *notification.Notifier
	Pepper     string
	RefreshTTL time.Duration
	OTPTTL     time.Duration
}

func New(d Deps) *Module {
	repo := NewRepository(d.DB)
	store := NewRedisStore(d.Redis)
	svc := NewService(repo, store, d.JWT, d.Notifier, d.Pepper, d.RefreshTTL, d.OTPTTL)
	return &Module{handler: NewHandler(svc), svc: svc, jwt: d.JWT}
}

// RequireAuth validates the bearer token against the tenant's secret.
func (m *Module) RequireAuth() gin.HandlerFunc {
	return middleware.RequireAuth(m.jwt, m.svc.ResolveSecret)
}

// RequirePrivilege gates a route on a named privilege (run after RequireAuth).
func (m *Module) RequirePrivilege(privilege string) gin.HandlerFunc {
	return middleware.RequirePrivilege(m.svc.HasPrivilege, privilege)
}

// HasPrivilege lets a module make a privilege decision inside a handler rather
// than as route middleware. The lecture module uses it to decide publish rights,
// so a join token's grants follow the same role definitions as every other check.
func (m *Module) HasPrivilege(ctx context.Context, userID, privilege string) (bool, error) {
	return m.svc.HasPrivilege(ctx, userID, privilege)
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/auth")
	{
		// Email verification comes first, then organization registration.
		g.POST("/send-verification", m.handler.SendVerification)
		g.POST("/resend-verification", m.handler.SendVerification) // alias
		g.POST("/verify-email", m.handler.VerifyEmail)
		g.POST("/register", m.handler.Register)

		g.POST("/login", m.handler.Login)
		g.POST("/refresh", m.handler.Refresh)
		g.POST("/forgot-password", m.handler.ForgotPassword)
		g.POST("/reset-password", m.handler.ResetPassword)

		g.POST("/logout", m.RequireAuth(), m.handler.Logout)
		g.POST("/change-password", m.RequireAuth(), m.handler.ChangePassword)
		g.GET("/me", m.RequireAuth(), m.handler.GetMe)
		g.GET("/privileges", m.RequireAuth(), m.handler.GetPrivileges)
	}
}
