package batches

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

// optQueryInt parses an optional positive-int query param, returning nil if absent/invalid.
func optQueryInt(c *gin.Context, key string) *int {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}

func optFormInt(c *gin.Context, key string) *int {
	v := c.PostForm(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}

func failErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not found")
	case errors.Is(err, ErrNameTaken):
		httpx.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrStorageUnavailable):
		httpx.Fail(c, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, ErrInvalidDateRange), errors.Is(err, ErrEmptyImport):
		httpx.Fail(c, http.StatusBadRequest, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, "something went wrong")
	}
}

// --- Batches -----------------------------------------------------------------

func (h *Handler) Create(c *gin.Context) {
	var req CreateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, userID := h.ids(c)
	b, err := h.svc.Create(c.Request.Context(), customerID, userID, req.CourseID, req.Name)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "batch created", b)
}

func (h *Handler) List(c *gin.Context) {
	customerID, _ := h.ids(c)
	page := httpx.ParsePage(c)
	res, err := h.svc.List(c.Request.Context(), customerID, optQueryInt(c, "course_id"), c.Query("status"), c.Query("search"), page)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", res)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	customerID, _ := h.ids(c)
	b, err := h.svc.Get(c.Request.Context(), customerID, id)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", b)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	var req UpdateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	b, err := h.svc.Update(c.Request.Context(), customerID, id, req.Name)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "batch updated", b)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.Delete(c.Request.Context(), customerID, id); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "batch deleted", nil)
}

func (h *Handler) Publish(c *gin.Context)   { h.setPublished(c, true) }
func (h *Handler) Unpublish(c *gin.Context) { h.setPublished(c, false) }

func (h *Handler) setPublished(c *gin.Context, published bool) {
	id, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	customerID, _ := h.ids(c)
	b, err := h.svc.SetPublished(c.Request.Context(), customerID, id, published)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "status updated", b)
}

// --- Module <-> tutor assignment ----------------------------------------------

func (h *Handler) AssignTutor(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	moduleID, ok := paramInt(c, "mid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid module id")
		return
	}
	var req AssignTutorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.AssignTutor(c.Request.Context(), customerID, batchID, moduleID, req); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "tutor assigned", nil)
}

func (h *Handler) UnassignTutor(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	moduleID, ok := paramInt(c, "mid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid module id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.UnassignTutor(c.Request.Context(), customerID, batchID, moduleID); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "tutor unassigned", nil)
}

func (h *Handler) ListTutors(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	customerID, _ := h.ids(c)
	tutors, err := h.svc.ListTutors(c.Request.Context(), customerID, batchID)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", tutors)
}

// --- Students ------------------------------------------------------------------

func (h *Handler) ListStudents(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	customerID, _ := h.ids(c)
	res, err := h.svc.ListStudents(c.Request.Context(), customerID, batchID, httpx.ParsePage(c))
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", res)
}

func (h *Handler) RemoveStudent(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	studentID, ok := paramInt(c, "sid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid student id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.RemoveStudent(c.Request.Context(), customerID, batchID, studentID); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "student removed", nil)
}

func (h *Handler) ImportStudents(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "a CSV file is required")
		return
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "could not read upload")
		return
	}
	defer f.Close()

	customerID, _ := h.ids(c)
	result, err := h.svc.ImportStudentsCSV(c.Request.Context(), customerID, batchID, f)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "import complete", result)
}

// --- Drive -----------------------------------------------------------------

func (h *Handler) ListDrive(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	customerID, _ := h.ids(c)
	nodes, err := h.svc.ListDrive(c.Request.Context(), customerID, batchID, optQueryInt(c, "parent_id"))
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "ok", nodes)
}

func (h *Handler) CreateFolder(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, userID := h.ids(c)
	node, err := h.svc.CreateFolder(c.Request.Context(), customerID, userID, batchID, req.Name, req.ParentID)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "folder created", node)
}

func (h *Handler) UploadFile(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "a file is required")
		return
	}
	customerID, userID := h.ids(c)
	node, err := h.svc.UploadFile(c.Request.Context(), customerID, userID, batchID, optFormInt(c, "parent_id"), fh)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, "file uploaded", node)
}

func (h *Handler) RenameNode(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	nodeID, ok := paramInt(c, "nid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid node id")
		return
	}
	var req RenameNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customerID, _ := h.ids(c)
	node, err := h.svc.RenameNode(c.Request.Context(), customerID, batchID, nodeID, req.Name)
	if err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "renamed", node)
}

func (h *Handler) DeleteNode(c *gin.Context) {
	batchID, ok := paramInt(c, "id")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid batch id")
		return
	}
	nodeID, ok := paramInt(c, "nid")
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid node id")
		return
	}
	customerID, _ := h.ids(c)
	if err := h.svc.DeleteNode(c.Request.Context(), customerID, batchID, nodeID); err != nil {
		failErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, "deleted", nil)
}
