package lecture

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tutorpilot/internal/middleware"
	"tutorpilot/internal/pkg/httpx"
	"tutorpilot/internal/pkg/scope"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// userID reads the caller's dashboard_users.id the same way every other module
// does: middleware only exposes it as a raw context string (CtxUserID), there is
// no centralized helper to parse it.
func userID(c *gin.Context) int {
	n, _ := strconv.Atoi(c.GetString(middleware.CtxUserID))
	return n
}

func lectureID(c *gin.Context) (int64, bool) {
	n, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func failErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not found")
	case errors.Is(err, ErrInvalidTransition):
		httpx.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotLive):
		httpx.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNoRecording):
		httpx.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrLiveKitUnavailable):
		httpx.Fail(c, http.StatusServiceUnavailable, err.Error())
	default:
		log.Printf("lecture: unhandled error: %v", err)
		httpx.Fail(c, http.StatusInternalServerError, "lecture operation failed")
	}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateLectureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	l, err := h.svc.Create(c.Request.Context(), scope.FromContext(c), userID(c), req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "lecture created", l)
}

func (h *Handler) List(c *gin.Context) {
	f := ListLectureFilter{
		Status: c.Query("status"),
		Search: c.Query("search"),
	}
	raw := c.Query("batch_id")
	if raw == "" {
		raw = c.Query("batchId")
	}
	if raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			f.BatchID = &n
		}
	}

	res, err := h.svc.List(c.Request.Context(), scope.FromContext(c), f, httpx.ParsePage(c))
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", res)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	l, err := h.svc.Get(c.Request.Context(), scope.FromContext(c), userID(c), id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", l)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	var req UpdateLectureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	l, err := h.svc.Update(c.Request.Context(), scope.FromContext(c), id, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture updated", l)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), scope.FromContext(c), id); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture deleted", nil)
}

func (h *Handler) Start(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	l, err := h.svc.Start(c.Request.Context(), scope.FromContext(c), id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture started", l)
}

func (h *Handler) End(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	l, err := h.svc.End(c.Request.Context(), scope.FromContext(c), id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture ended", l)
}

func (h *Handler) Cancel(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	l, err := h.svc.Cancel(c.Request.Context(), scope.FromContext(c), id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lecture cancelled", l)
}

func (h *Handler) Join(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	res, err := h.svc.Join(c.Request.Context(), scope.FromContext(c), userID(c), id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "joined", res)
}

func (h *Handler) Attendance(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	rows, err := h.svc.Attendance(c.Request.Context(), scope.FromContext(c), id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", rows)
}

func (h *Handler) Recording(c *gin.Context) {
	id, ok := lectureID(c)
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lecture id")
		return
	}
	url, err := h.svc.RecordingURL(c.Request.Context(), scope.FromContext(c), id)
	if err != nil {
		failErr(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}
