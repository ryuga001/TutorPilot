package auth

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tutorpilot/internal/middleware"
	dto "tutorpilot/internal/modules/auth/dto"
	model "tutorpilot/internal/modules/auth/model"
	service "tutorpilot/internal/modules/auth/service"
	"tutorpilot/internal/pkg/httpx"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrEmailNotVerified):
			httpx.Fail(c, http.StatusForbidden, err.Error())
		case errors.Is(err, model.ErrEmailTaken), errors.Is(err, model.ErrOrgTaken):
			httpx.Fail(c, http.StatusConflict, err.Error())
		default:
			httpx.Fail(c, http.StatusInternalServerError, "could not register user")
		}
		return
	}
	httpx.OK(c, http.StatusCreated, "registered", res)
}

func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not log in")
		return
	}
	httpx.OK(c, http.StatusOK, "logged in", res)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tokens, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, model.ErrInvalidToken) {
			httpx.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not refresh token")
		return
	}
	httpx.OK(c, http.StatusOK, "token refreshed", tokens)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
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

func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		if errors.Is(err, model.ErrInvalidOTP) {
			httpx.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not reset password")
		return
	}
	httpx.OK(c, http.StatusOK, "password updated", nil)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
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
		if errors.Is(err, model.ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not change password")
		return
	}
	httpx.OK(c, http.StatusOK, "password changed", res)
}

func (h *Handler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
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

func (h *Handler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.VerifyEmail(c.Request.Context(), req.Email, req.OTP); err != nil {
		if errors.Is(err, model.ErrInvalidOTP) {
			httpx.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not verify email")
		return
	}
	httpx.OK(c, http.StatusOK, "email verified", nil)
}

func (h *Handler) SendVerification(c *gin.Context) {
	var req dto.SendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.SendVerification(c.Request.Context(), req.Email); err != nil {
		if errors.Is(err, model.ErrEmailTaken) {
			httpx.Fail(c, http.StatusConflict, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "could not send verification code")
		return
	}
	httpx.OK(c, http.StatusOK, "verification code sent", nil)
}

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
