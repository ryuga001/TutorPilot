package server

import (
	"net/http"
	"tutorpilot/internal/config"
	"tutorpilot/internal/middleware"
	"tutorpilot/internal/modules/auth"
	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/pkg/jwtutil"
	"tutorpilot/internal/pkg/mailer"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.CORSAllowedOrigins))

	// Shared infrastructure.
	jwtMgr := jwtutil.New(cfg.JWTAccessTTL)
	mail := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)
	templates := notification.NewTemplateStore(db)
	notifier := notification.New(mail, templates, cfg.AppVerifyURL, cfg.OTPTTL, notification.SystemTenantID)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if cfg.AppEnv != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := r.Group("/api/v1")

	authModule := auth.New(auth.Deps{
		DB:         db,
		Redis:      rdb,
		JWT:        jwtMgr,
		Notifier:   notifier,
		Pepper:     cfg.PasswordPepper,
		RefreshTTL: cfg.JWTRefreshTTL,
		OTPTTL:     cfg.OTPTTL,
	})
	authModule.RegisterRoutes(api)

	return r
}
