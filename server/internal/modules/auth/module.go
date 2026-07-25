package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"workflow/internal/middleware"
	"workflow/internal/modules/notification"
	"workflow/internal/pkg/jwtutil"
)

type Module struct {
	handler *Handler
	jwt     *jwtutil.Manager
}

type Deps struct {
	DB         *pgxpool.Pool
	Redis      *redis.Client
	JWT        *jwtutil.Manager
	Notifier   *notification.Notifier
	Pepper     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	OTPTTL     time.Duration
}

func New(d Deps) *Module {
	repo := NewRepository(d.DB)
	store := NewRedisStore(d.Redis)
	svc := NewService(repo, store, d.JWT, d.Notifier, d.Pepper,
		d.AccessTTL, d.RefreshTTL, d.OTPTTL)
	return &Module{handler: NewHandler(svc), jwt: d.JWT}
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/auth")
	{
		// Email verification comes first, then register.
		g.POST("/send-verification", m.handler.SendVerification)
		g.POST("/resend-verification", m.handler.SendVerification) // alias
		g.POST("/verify-email", m.handler.VerifyEmail)
		g.POST("/register", m.handler.Register)

		g.POST("/login", m.handler.Login)
		g.POST("/refresh", m.handler.Refresh)
		g.POST("/forgot-password", m.handler.ForgotPassword)
		g.POST("/reset-password", m.handler.ResetPassword)

		g.POST("/logout", middleware.RequireAuth(m.jwt), m.handler.Logout)
		g.GET("/me", middleware.RequireAuth(m.jwt), m.handler.GetMe)
	}
}
