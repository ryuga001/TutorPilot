package auth

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"tutorpilot/internal/middleware"
	handler "tutorpilot/internal/modules/auth/handler"
	repository "tutorpilot/internal/modules/auth/repository"
	service "tutorpilot/internal/modules/auth/service"
	"tutorpilot/internal/pkg/jwtutil"
)

type Module struct {
	handler *handler.Handler
	svc     *service.Service
	jwt     *jwtutil.Manager
}

type Deps struct {
	DB         *pgxpool.Pool
	Redis      *redis.Client
	JWT        *jwtutil.Manager
	Pepper     string
	RefreshTTL time.Duration
	OTPTTL     time.Duration

	Stream    string
	VerifyURL string
}

func New(d Deps) *Module {
	repo := repository.NewRepository(d.DB)
	store := repository.NewRedisStore(d.Redis)
	svc := service.NewService(repo, store, d.JWT, d.DB, d.Pepper, d.RefreshTTL, d.OTPTTL, d.Stream, d.VerifyURL)
	return &Module{handler: handler.NewHandler(svc), svc: svc, jwt: d.JWT}
}

func (m *Module) RequireAuth() gin.HandlerFunc {
	return middleware.RequireAuth(m.jwt, m.svc.ResolveSecret)
}

func (m *Module) RequirePrivilege(privilege string) gin.HandlerFunc {
	return middleware.RequirePrivilege(m.svc.HasPrivilege, privilege)
}

func (m *Module) HasPrivilege(ctx context.Context, userID, privilege string) (bool, error) {
	return m.svc.HasPrivilege(ctx, userID, privilege)
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/auth")
	{
		g.POST("/send-verification", m.handler.SendVerification)
		g.POST("/resend-verification", m.handler.SendVerification)
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
