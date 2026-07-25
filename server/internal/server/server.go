package server

import (
	"net/http"
	"workflow/internal/config"
	"workflow/internal/middleware"
	"workflow/internal/modules/auth"
	"workflow/internal/modules/notification"
	"workflow/internal/pkg/jwtutil"
	"workflow/internal/pkg/mailer"

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
	jwtMgr := jwtutil.New(cfg.JWTSecret, cfg.JWTAccessTTL)
	mail := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)
	notifier := notification.New(mail, cfg.AppVerifyURL, cfg.OTPTTL)

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
		AccessTTL:  cfg.JWTAccessTTL,
		RefreshTTL: cfg.JWTRefreshTTL,
		OTPTTL:     cfg.OTPTTL,
	})
	authModule.RegisterRoutes(api)

	return r
}
