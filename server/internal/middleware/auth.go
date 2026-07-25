package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"workflow/internal/pkg/httpx"
	"workflow/internal/pkg/jwtutil"
)

const (
	CtxUserID = "userID"
	CtxEmail  = "email"
)

func RequireAuth(jwtMgr *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			httpx.Fail(c, http.StatusUnauthorized, "authorization header missing")
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httpx.Fail(c, http.StatusUnauthorized, "invalid authorization header")
			return
		}

		claims, err := jwtMgr.Parse(strings.TrimSpace(parts[1]))
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxEmail, claims.Email)
		c.Next()
	}
}
