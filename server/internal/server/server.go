package server

import (
	"log"
	"net/http"
	"tutorpilot/internal/modules/admin/livekit"

	"tutorpilot/internal/config"
	"tutorpilot/internal/middleware"
	batches "tutorpilot/internal/modules/admin/module/batches"
	courses "tutorpilot/internal/modules/admin/module/courses"
	lecture "tutorpilot/internal/modules/admin/module/lecture"
	students "tutorpilot/internal/modules/admin/module/students"
	tutors "tutorpilot/internal/modules/admin/module/tutors"
	webhooks "tutorpilot/internal/modules/admin/module/webhooks"
	"tutorpilot/internal/modules/admin/storage"
	auth "tutorpilot/internal/modules/auth/module"
	"tutorpilot/internal/pkg/jwtutil"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client, lkt *livekit.LiveKitClient) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.CORSAllowedOrigins))

	jwtMgr := jwtutil.New(cfg.JWTAccessTTL)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if cfg.AppEnv != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	authModule := auth.New(auth.Deps{
		DB:         db,
		Redis:      rdb,
		JWT:        jwtMgr,
		Pepper:     cfg.PasswordPepper,
		RefreshTTL: cfg.JWTRefreshTTL,
		OTPTTL:     cfg.OTPTTL,
		Stream:     cfg.EventStreamAuth,
		VerifyURL:  cfg.AppVerifyURL,
	})

	api := r.Group("/api/v1")

	authModule.RegisterRoutes(api)

	store, err := storage.New(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey,
		cfg.MinIOBucket, cfg.MinIOPublicURL, cfg.MinIOUseSSL)
	if err != nil {
		log.Printf("storage: MinIO unavailable, course uploads disabled: %v", err)
		store = nil
	}
	coursesModule := courses.New(courses.Deps{
		DB:          db,
		Storage:     store,
		RequireAuth: authModule.RequireAuth,
		RequirePriv: authModule.RequirePrivilege,
	})
	coursesModule.RegisterRoutes(api)

	tutorsModule := tutors.New(tutors.Deps{
		DB:          db,
		Storage:     store,
		Pepper:      cfg.PasswordPepper,
		SignInURL:   cfg.AppSignInURL,
		Stream:      cfg.EventStreamNotifications,
		RequireAuth: authModule.RequireAuth,
		RequirePriv: authModule.RequirePrivilege,
	})
	tutorsModule.RegisterRoutes(api)

	studentsModule := students.New(students.Deps{
		DB:          db,
		Storage:     store,
		Pepper:      cfg.PasswordPepper,
		SignInURL:   cfg.AppSignInURL,
		Stream:      cfg.EventStreamNotifications,
		RequireAuth: authModule.RequireAuth,
		RequirePriv: authModule.RequirePrivilege,
	})
	studentsModule.RegisterRoutes(api)

	batchesModule := batches.New(batches.Deps{
		DB:            db,
		Storage:       store,
		Pepper:        cfg.PasswordPepper,
		SignInURL:     cfg.AppSignInURL,
		Stream:        cfg.EventStreamNotifications,
		ImportMaxRows: cfg.ImportMaxRows,
		RequireAuth:   authModule.RequireAuth,
		RequirePriv:   authModule.RequirePrivilege,
	})
	batchesModule.RegisterRoutes(api)

	lecturesModule := lecture.New(lecture.Deps{
		DB:           db,
		LiveKit:      lkt,
		Storage:      store,
		HasPrivilege: authModule.HasPrivilege,
		JoinTokenTTL: cfg.LectureJoinTokenTTL,
		RequireAuth:  authModule.RequireAuth,
		RequirePriv:  authModule.RequirePrivilege,
	})

	if store != nil {
		lecturesModule.Service().SetDrive(batchesModule.DriveWriter(), store.PublicURL)
	}
	lecturesModule.RegisterRoutes(api)

	webhooks.New(webhooks.Deps{
		Lectures:  lecturesModule.Service(),
		APIKey:    cfg.LiveKitKey,
		APISecret: cfg.LiveKitSecret,
	}).RegisterRoutes(api)

	return r
}
