package server

import (
	"log"
	"net/http"
	"tutorpilot/internal/livekit"

	"tutorpilot/internal/config"
	"tutorpilot/internal/middleware"
	"tutorpilot/internal/modules/auth"
	"tutorpilot/internal/modules/batches"
	"tutorpilot/internal/modules/courses"
	"tutorpilot/internal/modules/lecture"
	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/modules/students"
	"tutorpilot/internal/modules/tutors"
	"tutorpilot/internal/modules/webhooks"
	"tutorpilot/internal/pkg/jwtutil"
	"tutorpilot/internal/pkg/mailer"
	"tutorpilot/internal/pkg/storage"

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
	mail := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)
	templates := notification.NewTemplateStore(db)
	notifier := notification.New(mail, templates, notification.Config{
		VerifyURL: cfg.AppVerifyURL,
		// Tutors/students no longer go through this notifier's invite templates
		// (see tutors/students Service.sendInvite, which emails the temporary
		// password directly), so a portal link template is not needed here.
		OTPTTL:         cfg.OTPTTL,
		InviteTTL:      cfg.InviteTTL,
		SystemTenantID: notification.SystemTenantID,
	})

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
		Notifier:   notifier,
		Pepper:     cfg.PasswordPepper,
		RefreshTTL: cfg.JWTRefreshTTL,
		OTPTTL:     cfg.OTPTTL,
	})

	// No host-based tenant resolution: the organization's subdomain is cosmetic
	// at the HTTP layer, handled entirely by middleware.CORS allowing the
	// wildcard origin. Every principal (admin, tutor, student) resolves the same
	// way, by customer_id on their dashboard_users row.
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

	// Creating a tutor or student also creates the dashboard_users login they
	// sign in with (see tutors/students Repository.Create), so each module needs
	// the password pepper and a mailer to send the temporary credentials.
	tutorsModule := tutors.New(tutors.Deps{
		DB:          db,
		Storage:     store,
		Mailer:      mail,
		Pepper:      cfg.PasswordPepper,
		RequireAuth: authModule.RequireAuth,
		RequirePriv: authModule.RequirePrivilege,
	})
	tutorsModule.RegisterRoutes(api)

	studentsModule := students.New(students.Deps{
		DB:          db,
		Storage:     store,
		Mailer:      mail,
		Pepper:      cfg.PasswordPepper,
		RequireAuth: authModule.RequireAuth,
		RequirePriv: authModule.RequirePrivilege,
	})
	studentsModule.RegisterRoutes(api)

	batchesModule := batches.New(batches.Deps{
		DB:          db,
		Storage:     store,
		Notifier:    notifier,
		Mailer:      mail,
		Pepper:      cfg.PasswordPepper,
		RequireAuth: authModule.RequireAuth,
		RequirePriv: authModule.RequirePrivilege,
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

	// Recordings are filed into the batch drive, which the batches module owns, so
	// the lecture service borrows just the two writes it needs.
	if store != nil {
		lecturesModule.Service().SetDrive(batchesModule.DriveWriter(), store.PublicURL)
	}
	lecturesModule.RegisterRoutes(api)

	// LiveKit's callbacks: they carry a signed token rather than a user session, so
	// they sit outside RequireAuth.
	webhooks.New(webhooks.Deps{
		Lectures:  lecturesModule.Service(),
		APIKey:    cfg.LiveKitKey,
		APISecret: cfg.LiveKitSecret,
	}).RegisterRoutes(api)

	return r
}
