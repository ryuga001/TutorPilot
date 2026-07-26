package lecture

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
	return &Handler{
		svc: svc,
	}
}

func (h *Handler) ids(c *gin.Context) (customerID, userID int) {
	customerID = middleware.CustomerID(c)
	userID, _ = strconv.Atoi(c.GetString(middleware.CtxUserID))
	return
}

func parseInt(c *gin.Context, key string) (int, bool) {
	n, err := strconv.Atoi(c.Param(key))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func failErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not found")
	default:
		if err != nil {
			httpx.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "lecture operation failed")
	}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateLectureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, userID := h.ids(c)
	lecture, err := h.svc.Create(c.Request.Context(), customerID, userID, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "lecture created", lecture)
}

func (h *Handler) List(c *gin.Context) {
	customerID, _ := h.ids(c)
	p := httpx.ParsePage(c)

	var batchID *int
	if bVal := c.Query("batchId"); bVal != "" {
		if val, err := strconv.Atoi(bVal); err == nil {
			batchID = &val
		}
	}

	req := ListLectureRequest{
		BatchID:  batchID,
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		Page:     p.Page,
		PageSize: p.PageSize,
	}

	res, err := h.svc.List(c.Request.Context(), customerID, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", res)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := parseInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	customerID, _ := h.ids(c)
	lecture, err := h.svc.Get(c.Request.Context(), customerID, id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", lecture)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	var req UpdateLectureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	lecture, err := h.svc.Update(c.Request.Context(), customerID, id, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture updated", lecture)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	customerID, _ := h.ids(c)
	err := h.svc.Delete(c.Request.Context(), customerID, id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture deleted", nil)
}

func (h *Handler) Start(c *gin.Context) {
	id, ok := parseInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	customerID, _ := h.ids(c)
	lecture, err := h.svc.Start(c.Request.Context(), customerID, id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture started", lecture)
}

func (h *Handler) End(c *gin.Context) {
	id, ok := parseInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	customerID, _ := h.ids(c)
	lecture, err := h.svc.End(c.Request.Context(), customerID, id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture ended", lecture)
}

func (h *Handler) Join(c *gin.Context) {
	id, ok := parseInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	customerID, _ := h.ids(c)
	userID := c.GetString(middleware.CtxUserID)
	email := c.GetString(middleware.CtxEmail)
	role := c.GetString(middleware.CtxRole)

	token, err := h.svc.Join(c.Request.Context(), customerID, userID, email, role, id)
	if err != nil {
		httpx.Fail(c, http.StatusForbidden, err.Error())
		return
	}

	httpx.OK(c, http.StatusOK, "joined", gin.H{"token": token})
}
