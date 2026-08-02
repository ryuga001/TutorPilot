package tutors

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tutorpilot/internal/middleware"
	dto "tutorpilot/internal/modules/admin/dto/tutors"
	model "tutorpilot/internal/modules/admin/model/tutors"
	service "tutorpilot/internal/modules/admin/service/tutors"
	"tutorpilot/internal/pkg/httpx"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ids(c *gin.Context) (customerID, userID int) {
	customerID = middleware.CustomerID(c)
	userID, _ = strconv.Atoi(c.GetString(middleware.CtxUserID))
	return
}

func paramInt(c *gin.Context, key string) (int, bool) {
	n, err := strconv.Atoi(c.Param(key))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func failErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not found")
	case errors.Is(err, model.ErrEmailTaken):
		httpx.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, model.ErrNoTutorRole):
		httpx.Fail(c, http.StatusInternalServerError, err.Error())
	case errors.Is(err, model.ErrStorageUnavailable):
		httpx.Fail(c, http.StatusServiceUnavailable, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, "something went wrong")
	}
}

func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateTutorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, userID := h.ids(c)
	t, err := h.svc.Create(c.Request.Context(), customerID, userID, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "tutor created", t)
}

func (h *Handler) List(c *gin.Context) {
	customerID, _ := h.ids(c)
	page := httpx.ParsePage(c)
	res, err := h.svc.List(c.Request.Context(), customerID, c.Query("search"), page)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", res)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid tutor id")
		return
	}
	customerID, _ := h.ids(c)
	t, err := h.svc.Get(c.Request.Context(), customerID, id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", t)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid tutor id")
		return
	}
	var req dto.UpdateTutorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	t, err := h.svc.Update(c.Request.Context(), customerID, id, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "tutor updated", t)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid tutor id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.Delete(c.Request.Context(), customerID, id); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "tutor deleted", nil)
}

func (h *Handler) UploadProfileImage(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid tutor id")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "a file is required")
		return
	}
	customerID, _ := h.ids(c)
	t, err := h.svc.UploadProfileImage(c.Request.Context(), customerID, id, fh)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "profile image updated", t)
}
