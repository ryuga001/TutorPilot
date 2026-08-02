package courses

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tutorpilot/internal/middleware"
	dto "tutorpilot/internal/modules/admin/dto/courses"
	model "tutorpilot/internal/modules/admin/model/courses"
	service "tutorpilot/internal/modules/admin/service/courses"
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
	case errors.Is(err, model.ErrSlugTaken):
		httpx.Fail(c, http.StatusConflict, "a course with a similar title already exists")
	case errors.Is(err, model.ErrStorageUnavailable):
		httpx.Fail(c, http.StatusServiceUnavailable, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, "something went wrong")
	}
}

func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, userID := h.ids(c)
	course, err := h.svc.Create(c.Request.Context(), customerID, userID, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "course created", course)
}

func (h *Handler) List(c *gin.Context) {
	customerID, _ := h.ids(c)
	page := httpx.ParsePage(c)
	res, err := h.svc.List(c.Request.Context(), customerID, c.Query("status"), c.Query("search"), page)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", res)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	customerID, _ := h.ids(c)
	course, err := h.svc.Get(c.Request.Context(), customerID, id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", course)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	var req dto.UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	course, err := h.svc.Update(c.Request.Context(), customerID, id, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "course updated", course)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.Delete(c.Request.Context(), customerID, id); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "course deleted", nil)
}

func (h *Handler) Publish(c *gin.Context)   { h.setPublished(c, true) }
func (h *Handler) Unpublish(c *gin.Context) { h.setPublished(c, false) }

func (h *Handler) setPublished(c *gin.Context, published bool) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	customerID, _ := h.ids(c)
	course, err := h.svc.SetPublished(c.Request.Context(), customerID, id, published)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "status updated", course)
}

func (h *Handler) CreateModule(c *gin.Context) {
	courseID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	var req dto.ModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	m, err := h.svc.CreateModule(c.Request.Context(), customerID, courseID, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "module created", m)
}

func (h *Handler) UpdateModule(c *gin.Context) {
	moduleID, ok := paramInt(c, "mid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid module id")
		return
	}
	var req dto.ModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.UpdateModule(c.Request.Context(), customerID, moduleID, req); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "module updated", nil)
}

func (h *Handler) DeleteModule(c *gin.Context) {
	moduleID, ok := paramInt(c, "mid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid module id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.DeleteModule(c.Request.Context(), customerID, moduleID); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "module deleted", nil)
}

func (h *Handler) CreateLesson(c *gin.Context) {
	moduleID, ok := paramInt(c, "mid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid module id")
		return
	}
	var req dto.LessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	l, err := h.svc.CreateLesson(c.Request.Context(), customerID, moduleID, req)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "lesson created", l)
}

func (h *Handler) UpdateLesson(c *gin.Context) {
	lessonID, ok := paramInt(c, "lid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lesson id")
		return
	}
	var req dto.LessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.UpdateLesson(c.Request.Context(), customerID, lessonID, req); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lesson updated", nil)
}

func (h *Handler) DeleteLesson(c *gin.Context) {
	lessonID, ok := paramInt(c, "lid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid lesson id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.DeleteLesson(c.Request.Context(), customerID, lessonID); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "lesson deleted", nil)
}

func (h *Handler) ListResources(c *gin.Context) {
	courseID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	customerID, _ := h.ids(c)
	res, err := h.svc.ListResources(c.Request.Context(), customerID, courseID, httpx.ParsePage(c))
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", res)
}

func (h *Handler) UploadResource(c *gin.Context) {
	courseID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "a file is required")
		return
	}
	var lessonID *int
	if v := c.PostForm("lesson_id"); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			lessonID = &n
		}
	}
	f, err := fileHeader.Open()
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "could not read upload")
		return
	}
	defer f.Close()

	customerID, userID := h.ids(c)
	res, err := h.svc.UploadResource(c.Request.Context(), customerID, courseID, userID, lessonID,
		fileHeader.Filename, fileHeader.Header.Get("Content-Type"), f, fileHeader.Size)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "uploaded", res)
}

func (h *Handler) DeleteResource(c *gin.Context) {
	courseID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	resourceID, ok := paramInt(c, "rid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid resource id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.DeleteResource(c.Request.Context(), customerID, courseID, resourceID); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "resource deleted", nil)
}

func (h *Handler) UploadThumbnail(c *gin.Context) {
	courseID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid course id")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "a file is required")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "could not read upload")
		return
	}
	defer f.Close()

	customerID, _ := h.ids(c)
	course, err := h.svc.UploadThumbnail(c.Request.Context(), customerID, courseID,
		fileHeader.Filename, fileHeader.Header.Get("Content-Type"), f, fileHeader.Size)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "thumbnail updated", course)
}
