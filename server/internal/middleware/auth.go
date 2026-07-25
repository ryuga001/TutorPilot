package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"tutorpilot/internal/pkg/httpx"
	"tutorpilot/internal/pkg/jwtutil"
)

const (
	CtxUserID     = "userID"
	CtxEmail      = "email"
	CtxCustomerID = "customerID"
	CtxRole       = "role"
)

// SecretResolver returns the signing secret for a tenant.
type SecretResolver func(ctx context.Context, customerID int) ([]byte, error)

// PrivilegeChecker reports whether a user holds a named privilege.
type PrivilegeChecker func(ctx context.Context, userID, privilege string) (bool, error)

// RequireAuth validates the bearer access token (verified against the tenant's
// secret) and stores the identity on the context.
func RequireAuth(jwtMgr *jwtutil.Manager, resolve SecretResolver) gin.HandlerFunc {
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

		claims, err := jwtMgr.Parse(strings.TrimSpace(parts[1]), func(customerID int) ([]byte, error) {
			return resolve(c.Request.Context(), customerID)
		})
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxEmail, claims.Email)
		c.Set(CtxCustomerID, claims.CustomerID)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}

// RequirePrivilege gates a route on a named privilege. Must run after RequireAuth.
func RequirePrivilege(check PrivilegeChecker, privilege string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(CtxUserID)
		if userID == "" {
			httpx.Fail(c, http.StatusUnauthorized, "not authenticated")
			return
		}
		ok, err := check(c.Request.Context(), userID, privilege)
		if err != nil {
			httpx.Fail(c, http.StatusInternalServerError, "could not verify privileges")
			return
		}
		if !ok {
			httpx.Fail(c, http.StatusForbidden, "missing required privilege: "+privilege)
			return
		}
		c.Next()
	}
}

// CustomerID reads the authenticated tenant id from the context.
func CustomerID(c *gin.Context) int {
	v, _ := c.Get(CtxCustomerID)
	switch id := v.(type) {
	case int:
		return id
	case string:
		n, _ := strconv.Atoi(id)
		return n
	default:
		return 0
	}
}
