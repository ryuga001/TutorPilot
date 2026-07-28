package auth

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tutorpilot/internal/middleware"
	"tutorpilot/internal/pkg/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register godoc
// @Summary      Register a verified email
// @Description  Creates the account for an already-verified email and returns tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Name, email, password"
// @Success      201      {object}  httpx.Envelope{data=AuthResponse}
// @Failure      400      {object}  httpx.Envelope
// @Failure      403      {object}  httpx.Envelope  "email not verified"
// @Failure      409      {object}  httpx.Envelope  "email already registered"
// @Router       /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailNotVerified):
			httpx.Fail(c, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrOrgTaken):
			httpx.Fail(c, http.StatusConflict, err.Error())
		default:
			httpx.Fail(c, http.StatusInternalServerError, "could not register user")
		}
		return
	}
	httpx.OK(c, http.StatusCreated, "registered", res)
}

// Login godoc
// @Summary      Log in
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Email and password"
// @Success      200      {object}  httpx.Envelope{data=AuthResponse}
// @Failure      401      {object}  httpx.Envelope
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not log in")
		return
	}
	httpx.OK(c, http.StatusOK, "logged in", res)
}

// Refresh godoc
// @Summary      Rotate refresh token
// @Description  Exchanges a valid refresh token for a new access + refresh pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RefreshRequest  true  "Refresh token"
// @Success      200      {object}  httpx.Envelope{data=TokenPair}
// @Failure      401      {object}  httpx.Envelope
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tokens, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			httpx.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not refresh token")
		return
	}
	httpx.OK(c, http.StatusOK, "token refreshed", tokens)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "could not process request")
		return
	}
	httpx.OK(c, http.StatusOK, "Reset code has been sent to the email", nil)
}

// ResetPassword godoc
// @Summary      Reset password with a code
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      ResetPasswordRequest  true  "Email, OTP, new password"
// @Success      200      {object}  httpx.Envelope
// @Failure      400      {object}  httpx.Envelope  "invalid or expired otp"
// @Router       /auth/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		if errors.Is(err, ErrInvalidOTP) {
			httpx.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not reset password")
		return
	}
	httpx.OK(c, http.StatusOK, "password updated", nil)
}

// ChangePassword godoc
// @Summary      Change your own password
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      ChangePasswordRequest  true  "Current and new password"
// @Success      200      {object}  httpx.Envelope{data=AuthResponse}
// @Failure      401      {object}  httpx.Envelope  "current password is wrong"
// @Router       /auth/change-password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := strconv.Atoi(c.GetString(middleware.CtxUserID))
	if err != nil {
		httpx.Fail(c, http.StatusUnauthorized, "invalid session")
		return
	}
	res, err := h.svc.ChangePassword(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not change password")
		return
	}
	httpx.OK(c, http.StatusOK, "password changed", res)
}

// Logout godoc
// @Summary      Log out (revoke a refresh token)
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      LogoutRequest  true  "Refresh token to revoke"
// @Success      200      {object}  httpx.Envelope
// @Failure      401      {object}  httpx.Envelope
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "could not log out")
		return
	}
	httpx.OK(c, http.StatusOK, "logged out", nil)
}

// VerifyEmail godoc
// @Summary      Verify an email OTP
// @Description  Confirms the code and marks the email verified (no user created yet).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      VerifyEmailRequest  true  "Email and 6-digit OTP"
// @Success      200      {object}  httpx.Envelope
// @Failure      400      {object}  httpx.Envelope  "invalid or expired otp"
// @Router       /auth/verify-email [post]
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.VerifyEmail(c.Request.Context(), req.Email, req.OTP); err != nil {
		if errors.Is(err, ErrInvalidOTP) {
			httpx.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not verify email")
		return
	}
	httpx.OK(c, http.StatusOK, "email verified", nil)
}

// SendVerification godoc
// @Summary      Send an email verification code
// @Description  Emails a 6-digit OTP for an unregistered email. Also mounted at /auth/resend-verification.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      SendVerificationRequest  true  "Email"
// @Success      200      {object}  httpx.Envelope
// @Failure      409      {object}  httpx.Envelope  "email already registered"
// @Router       /auth/send-verification [post]
// @Router       /auth/resend-verification [post]
func (h *Handler) SendVerification(c *gin.Context) {
	var req SendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.SendVerification(c.Request.Context(), req.Email); err != nil {
		if errors.Is(err, ErrEmailTaken) {
			httpx.Fail(c, http.StatusConflict, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not send verification code")
		return
	}
	httpx.OK(c, http.StatusOK, "verification code sent", nil)
}

// GetMe godoc
// @Summary      Current user
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  httpx.Envelope{data=UserView}
// @Failure      401  {object}  httpx.Envelope
// @Router       /auth/me [get]
// GetPrivileges returns the current user's privilege names (for frontend gating).
func (h *Handler) GetPrivileges(c *gin.Context) {
	userID, err := strconv.Atoi(c.GetString(middleware.CtxUserID))
	if err != nil {
		httpx.Fail(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	privs, err := h.svc.GetPrivileges(c.Request.Context(), userID)
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "could not load privileges")
		return
	}
	httpx.OK(c, http.StatusOK, "ok", privs)
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, err := strconv.Atoi(c.GetString(middleware.CtxUserID))
	if err != nil {
		httpx.Fail(c, http.StatusUnauthorized, "invalid session")
		return
	}
	user, err := h.svc.GetMe(c.Request.Context(), userID)
	if err != nil {
		httpx.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	httpx.OK(c, http.StatusOK, "ok", user)
}
