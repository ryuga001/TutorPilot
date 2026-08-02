package courses

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	dto "tutorpilot/internal/modules/admin/dto/courses"
	model "tutorpilot/internal/modules/admin/model/courses"
	repository "tutorpilot/internal/modules/admin/repository/courses"
	"tutorpilot/internal/modules/admin/storage"
	"tutorpilot/internal/pkg/httpx"
)

type Service struct {
	repo    *repository.Repository
	storage *storage.Storage
}

func NewService(repo *repository.Repository, store *storage.Storage) *Service {
	return &Service{repo: repo, storage: store}
}

func (s *Service) Create(ctx context.Context, customerID, userID int, req dto.CreateCourseRequest) (*model.CourseView, error) {
	base := slugify(req.Title)
	slug := base
	for i := 0; i < 6; i++ {
		c, err := s.repo.CreateCourse(ctx, customerID, userID, req.Title, slug, req.Summary, req.DescriptionMD)
		if errors.Is(err, model.ErrSlugTaken) {
			slug = fmt.Sprintf("%s-%d", base, i+2)
			continue
		}
		if err != nil {
			return nil, err
		}
		v := s.courseView(c)
		return &v, nil
	}
	return nil, model.ErrSlugTaken
}

func (s *Service) List(ctx context.Context, customerID int, status, search string, p httpx.Page) (httpx.Paginated[model.CourseView], error) {
	rows, total, err := s.repo.ListCourses(ctx, customerID, status, search, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[model.CourseView]{}, err
	}
	items := make([]model.CourseView, 0, len(rows))
	for i := range rows {
		items = append(items, s.courseView(&rows[i]))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (*model.CourseView, error) {
	c, err := s.repo.GetCourse(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	mods, err := s.repo.LoadModules(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	v := s.courseView(c)
	v.Modules = moduleViews(mods)
	return &v, nil
}

func (s *Service) Update(ctx context.Context, customerID, id int, req dto.UpdateCourseRequest) (*model.CourseView, error) {
	c, err := s.repo.UpdateCourse(ctx, customerID, id, req.Title, req.Summary, req.DescriptionMD)
	if err != nil {
		return nil, err
	}
	v := s.courseView(c)
	return &v, nil
}

func (s *Service) Delete(ctx context.Context, customerID, id int) error {
	return s.repo.DeleteCourse(ctx, customerID, id)
}

func (s *Service) SetPublished(ctx context.Context, customerID, id int, published bool) (*model.CourseView, error) {
	status := model.StatusDraft
	var at *time.Time
	if published {
		status = model.StatusPublished
		now := time.Now()
		at = &now
	}
	c, err := s.repo.SetStatus(ctx, customerID, id, status, at)
	if err != nil {
		return nil, err
	}
	v := s.courseView(c)
	return &v, nil
}

func (s *Service) CreateModule(ctx context.Context, customerID, courseID int, req dto.ModuleRequest) (*model.ModuleView, error) {
	if _, err := s.repo.GetCourse(ctx, customerID, courseID); err != nil {
		return nil, err
	}
	m, err := s.repo.CreateModule(ctx, courseID, req.Title, req.Position)
	if err != nil {
		return nil, err
	}
	v := moduleView(*m)
	return &v, nil
}

func (s *Service) UpdateModule(ctx context.Context, customerID, moduleID int, req dto.ModuleRequest) error {
	if _, err := s.repo.ModuleOwned(ctx, customerID, moduleID); err != nil {
		return err
	}
	return s.repo.UpdateModule(ctx, moduleID, req.Title, req.Position)
}

func (s *Service) DeleteModule(ctx context.Context, customerID, moduleID int) error {
	if _, err := s.repo.ModuleOwned(ctx, customerID, moduleID); err != nil {
		return err
	}
	return s.repo.DeleteModule(ctx, moduleID)
}

func (s *Service) CreateLesson(ctx context.Context, customerID, moduleID int, req dto.LessonRequest) (*model.LessonView, error) {
	if _, err := s.repo.ModuleOwned(ctx, customerID, moduleID); err != nil {
		return nil, err
	}
	l, err := s.repo.CreateLesson(ctx, moduleID, req.Title, req.ContentMD, req.Position)
	if err != nil {
		return nil, err
	}
	v := lessonView(*l)
	return &v, nil
}

func (s *Service) UpdateLesson(ctx context.Context, customerID, lessonID int, req dto.LessonRequest) error {
	if err := s.repo.LessonOwned(ctx, customerID, lessonID); err != nil {
		return err
	}
	return s.repo.UpdateLesson(ctx, lessonID, req.Title, req.ContentMD, req.Position)
}

func (s *Service) DeleteLesson(ctx context.Context, customerID, lessonID int) error {
	if err := s.repo.LessonOwned(ctx, customerID, lessonID); err != nil {
		return err
	}
	return s.repo.DeleteLesson(ctx, lessonID)
}

func (s *Service) ListResources(ctx context.Context, customerID, courseID int, p httpx.Page) (httpx.Paginated[model.ResourceView], error) {
	if _, err := s.repo.GetCourse(ctx, customerID, courseID); err != nil {
		return httpx.Paginated[model.ResourceView]{}, err
	}
	rows, total, err := s.repo.ListResources(ctx, courseID, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[model.ResourceView]{}, err
	}
	items := make([]model.ResourceView, 0, len(rows))
	for i := range rows {
		items = append(items, s.resourceView(&rows[i]))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) UploadResource(ctx context.Context, customerID, courseID, userID int, lessonID *int, name, contentType string, r io.Reader, size int64) (*model.ResourceView, error) {
	if s.storage == nil {
		return nil, model.ErrStorageUnavailable
	}
	if _, err := s.repo.GetCourse(ctx, customerID, courseID); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("customer_%d/courses/%d/resources/%s-%s", customerID, courseID, randToken(), sanitizeName(name))
	if err := s.storage.Upload(ctx, key, r, size, contentType); err != nil {
		return nil, err
	}
	res, err := s.repo.CreateResource(ctx, courseID, lessonID, userID, name, key, contentType, size)
	if err != nil {
		_ = s.storage.Remove(ctx, key)
		return nil, err
	}
	v := s.resourceView(res)
	return &v, nil
}

func (s *Service) DeleteResource(ctx context.Context, customerID, courseID, resourceID int) error {
	res, err := s.repo.GetResource(ctx, customerID, courseID, resourceID)
	if err != nil {
		return err
	}
	if s.storage != nil {
		_ = s.storage.Remove(ctx, res.ObjectKey)
	}
	return s.repo.DeleteResource(ctx, res.ID)
}

func (s *Service) UploadThumbnail(ctx context.Context, customerID, courseID int, name, contentType string, r io.Reader, size int64) (*model.CourseView, error) {
	if s.storage == nil {
		return nil, model.ErrStorageUnavailable
	}
	current, err := s.repo.GetCourse(ctx, customerID, courseID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("customer_%d/courses/%d/thumbnail/%s-%s", customerID, courseID, randToken(), sanitizeName(name))
	if err := s.storage.Upload(ctx, key, r, size, contentType); err != nil {
		return nil, err
	}
	c, err := s.repo.SetThumbnail(ctx, customerID, courseID, key)
	if err != nil {
		_ = s.storage.Remove(ctx, key)
		return nil, err
	}
	if current.ThumbnailKey != nil && *current.ThumbnailKey != "" {
		_ = s.storage.Remove(ctx, *current.ThumbnailKey)
	}
	v := s.courseView(c)
	return &v, nil
}

func (s *Service) courseView(c *model.Course) model.CourseView {
	v := model.CourseView{
		ID: c.ID, Title: c.Title, Slug: c.Slug, Summary: c.Summary,
		DescriptionMD: c.DescriptionMD, Status: c.Status, PublishedAt: c.PublishedAt,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if s.storage != nil && c.ThumbnailKey != nil && *c.ThumbnailKey != "" {
		v.ThumbnailURL = s.storage.PublicURL(*c.ThumbnailKey)
	}
	return v
}

func (s *Service) resourceView(r *model.Resource) model.ResourceView {
	url := ""
	if s.storage != nil {
		url = s.storage.PublicURL(r.ObjectKey)
	}
	return model.ResourceView{
		ID: r.ID, LessonID: r.LessonID, Name: r.Name, URL: url,
		ContentType: r.ContentType, SizeBytes: r.SizeBytes, CreatedAt: r.CreatedAt,
	}
}

func moduleViews(mods []model.CourseModule) []model.ModuleView {
	out := make([]model.ModuleView, 0, len(mods))
	for _, m := range mods {
		out = append(out, moduleView(m))
	}
	return out
}

func moduleView(m model.CourseModule) model.ModuleView {
	lessons := make([]model.LessonView, 0, len(m.Lessons))
	for _, l := range m.Lessons {
		lessons = append(lessons, lessonView(l))
	}
	return model.ModuleView{ID: m.ID, Title: m.Title, Position: m.Position, Lessons: lessons}
}

func lessonView(l model.Lesson) model.LessonView {
	return model.LessonView{ID: l.ID, Title: l.Title, ContentMD: l.ContentMD, Position: l.Position}
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "course"
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func sanitizeName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.ReplaceAll(name, " ", "-")
	if name == "" || name == "." || name == "/" {
		name = "file"
	}
	return name
}

func randToken() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
