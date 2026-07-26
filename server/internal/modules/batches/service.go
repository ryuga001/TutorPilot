package batches

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"

	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/pkg/httpx"
	"tutorpilot/internal/pkg/storage"
)

var (
	ErrStorageUnavailable = errors.New("file storage is not configured")
	ErrInvalidDateRange   = errors.New("expected_end_date must be on or after start_date")
	ErrEmptyImport        = errors.New("no valid rows found in the uploaded file")
)

const dateLayout = "2006-01-02"

type Service struct {
	repo     *Repository
	storage  *storage.Storage
	notifier *notification.Notifier
}

func NewService(repo *Repository, store *storage.Storage, notifier *notification.Notifier) *Service {
	return &Service{repo: repo, storage: store, notifier: notifier}
}

func (s *Service) Create(ctx context.Context, customerID, userID, courseID int, name string) (*BatchView, error) {
	b, err := s.repo.CreateBatch(ctx, customerID, userID, courseID, name)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, b)
}

func (s *Service) List(ctx context.Context, customerID int, courseID *int, status, search string, p httpx.Page) (httpx.Paginated[BatchView], error) {
	rows, total, err := s.repo.ListBatches(ctx, customerID, courseID, status, search, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[BatchView]{}, err
	}
	items := make([]BatchView, 0, len(rows))
	for i := range rows {
		v, err := s.view(ctx, &rows[i])
		if err != nil {
			return httpx.Paginated[BatchView]{}, err
		}
		items = append(items, *v)
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (*BatchView, error) {
	b, err := s.repo.GetBatch(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, b)
}

func (s *Service) Update(ctx context.Context, customerID, id int, name string) (*BatchView, error) {
	b, err := s.repo.UpdateBatch(ctx, customerID, id, name)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, b)
}

func (s *Service) Delete(ctx context.Context, customerID, id int) error {
	return s.repo.DeleteBatch(ctx, customerID, id)
}

func (s *Service) SetPublished(ctx context.Context, customerID, id int, published bool) (*BatchView, error) {
	status := StatusDraft
	var at *time.Time
	if published {
		status = StatusPublished
		now := time.Now()
		at = &now
	}
	b, err := s.repo.SetStatus(ctx, customerID, id, status, at)
	if err != nil {
		return nil, err
	}

	if published {
		s.notifyPublish(ctx, b)
	}

	return s.view(ctx, b)
}

func (s *Service) notifyPublish(ctx context.Context, b *Batch) {
	courseTitle, err := s.repo.CourseTitle(ctx, b.CourseID)
	if err != nil {
		log.Printf("batches: could not load course title for batch %d: %v", b.ID, err)
		courseTitle = ""
	}

	assignments, err := s.repo.LoadModuleAssignments(ctx, b.ID, b.CourseID)
	if err != nil {
		log.Printf("batches: could not load module assignments for batch %d: %v", b.ID, err)
	}
	for _, a := range assignments {
		if a.TutorID == nil || a.TutorEmail == nil {
			continue
		}
		vars := map[string]string{
			"name":         firstNonEmpty(a.TutorFirstName),
			"batch_name":   b.Name,
			"course_name":  courseTitle,
			"module_title": a.ModuleTitle,
		}
		if a.StartDate != nil {
			vars["start_date"] = a.StartDate.Format(dateLayout)
		}
		if a.ExpectedEndDate != nil {
			vars["expected_end_date"] = a.ExpectedEndDate.Format(dateLayout)
		}
		if err := s.notifier.SendBatchTutorAssignment(ctx, *a.TutorEmail, vars); err != nil {
			log.Printf("batches: failed to email tutor %s for batch %d: %v", *a.TutorEmail, b.ID, err)
		}
	}

	students, err := s.repo.LoadAllStudents(ctx, b.ID)
	if err != nil {
		log.Printf("batches: could not load students for batch %d: %v", b.ID, err)
		return
	}
	for _, st := range students {
		vars := map[string]string{
			"name":        st.FirstName,
			"batch_name":  b.Name,
			"course_name": courseTitle,
		}
		if err := s.notifier.SendBatchStudentEnrollment(ctx, st.Email, vars); err != nil {
			log.Printf("batches: failed to email student %s for batch %d: %v", st.Email, b.ID, err)
		}
	}
}

func firstNonEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Service) AssignTutor(ctx context.Context, customerID, batchID, courseModuleID int, req AssignTutorRequest) error {
	start, err := time.Parse(dateLayout, req.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start_date, expected YYYY-MM-DD")
	}
	end, err := time.Parse(dateLayout, req.ExpectedEndDate)
	if err != nil {
		return fmt.Errorf("invalid expected_end_date, expected YYYY-MM-DD")
	}
	if end.Before(start) {
		return ErrInvalidDateRange
	}
	batch, err := s.repo.GetBatch(ctx, customerID, batchID)
	if err != nil {
		return err
	}
	return s.repo.AssignModuleTutor(ctx, customerID, batchID, batch.CourseID, courseModuleID, req.TutorID, start, end)
}

func (s *Service) UnassignTutor(ctx context.Context, customerID, batchID, courseModuleID int) error {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return err
	}
	return s.repo.UnassignModuleTutor(ctx, batchID, courseModuleID)
}

func (s *Service) ListTutors(ctx context.Context, customerID, batchID int) ([]TutorSummaryView, error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}
	rows, err := s.repo.LoadTutors(ctx, batchID)
	if err != nil {
		return nil, err
	}
	out := make([]TutorSummaryView, 0, len(rows))
	for _, t := range rows {
		out = append(out, tutorSummaryView(t))
	}
	return out, nil
}

func (s *Service) ListStudents(ctx context.Context, customerID, batchID int, p httpx.Page) (httpx.Paginated[StudentSummaryView], error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return httpx.Paginated[StudentSummaryView]{}, err
	}
	rows, total, err := s.repo.LoadStudents(ctx, batchID, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[StudentSummaryView]{}, err
	}
	items := make([]StudentSummaryView, 0, len(rows))
	for _, st := range rows {
		items = append(items, studentSummaryView(st))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) EnrollStudents(ctx context.Context, customerID, batchID int, studentIDs []int) (*EnrollResult, error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}
	enrolled, notFound, err := s.repo.EnrollStudentIDs(ctx, customerID, batchID, studentIDs)
	if err != nil {
		return nil, err
	}
	return &EnrollResult{Enrolled: enrolled, NotFound: notFound}, nil
}

func (s *Service) RemoveStudent(ctx context.Context, customerID, batchID, studentID int) error {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return err
	}
	return s.repo.RemoveStudent(ctx, batchID, studentID)
}

func (s *Service) ImportStudentsCSV(ctx context.Context, customerID, batchID int, r io.Reader) (*ImportResult, error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}

	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("could not read CSV header: %w", err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	firstIdx, hasFirst := col["first_name"]
	lastIdx, hasLast := col["last_name"]
	emailIdx, hasEmail := col["email"]
	phoneIdx, hasPhone := col["phone"]
	if !hasPhone {
		phoneIdx, hasPhone = col["phone_no"]
	}
	if !hasFirst || !hasLast || !hasEmail {
		return nil, fmt.Errorf("CSV header must include first_name, last_name, and email columns")
	}

	valid := make([]StudentRow, 0)
	skipped := make([]SkippedRow, 0)
	rowNum := 1 // header is row 1

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		rowNum++
		if err != nil {
			skipped = append(skipped, SkippedRow{Row: rowNum, Reason: "could not parse row"})
			continue
		}

		field := func(idx int, ok bool) string {
			if !ok || idx >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[idx])
		}

		first := field(firstIdx, hasFirst)
		last := field(lastIdx, hasLast)
		email := strings.ToLower(field(emailIdx, hasEmail))
		phone := field(phoneIdx, hasPhone)

		if first == "" || last == "" {
			skipped = append(skipped, SkippedRow{Row: rowNum, Reason: "missing first or last name"})
			continue
		}
		if _, err := mail.ParseAddress(email); err != nil {
			skipped = append(skipped, SkippedRow{Row: rowNum, Reason: "invalid email"})
			continue
		}
		valid = append(valid, StudentRow{FirstName: first, LastName: last, Email: email, Phone: phone})
	}

	imported := 0
	if len(valid) > 0 {
		imported, err = s.repo.ImportStudents(ctx, customerID, batchID, valid)
		if err != nil {
			return nil, err
		}
	}
	if imported == 0 && len(skipped) == 0 {
		return nil, ErrEmptyImport
	}

	return &ImportResult{Imported: imported, Skipped: skipped}, nil
}

func (s *Service) ListDrive(ctx context.Context, customerID, batchID int, parentID *int) ([]DriveNodeView, error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListChildren(ctx, batchID, parentID)
	if err != nil {
		return nil, err
	}
	out := make([]DriveNodeView, 0, len(rows))
	for i := range rows {
		out = append(out, s.nodeView(&rows[i]))
	}
	return out, nil
}

func (s *Service) CreateFolder(ctx context.Context, customerID, userID, batchID int, name string, parentID *int) (*DriveNodeView, error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}
	n, err := s.repo.CreateFolder(ctx, batchID, parentID, name, userID)
	if err != nil {
		return nil, err
	}
	v := s.nodeView(n)
	return &v, nil
}

func (s *Service) UploadFile(ctx context.Context, customerID, userID, batchID int, parentID *int, fh *multipart.FileHeader) (*DriveNodeView, error) {
	if s.storage == nil {
		return nil, ErrStorageUnavailable
	}
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("customer_%d/batches/%d/drive", customerID, batchID)
	key, err := s.storage.UploadFile(ctx, prefix, fh)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.CreateFile(ctx, batchID, parentID, fh.Filename, key, fh.Header.Get("Content-Type"), fh.Size, userID)
	if err != nil {
		_ = s.storage.Remove(ctx, key)
		return nil, err
	}
	v := s.nodeView(n)
	return &v, nil
}

func (s *Service) RenameNode(ctx context.Context, customerID, batchID, nodeID int, name string) (*DriveNodeView, error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}
	n, err := s.repo.RenameNode(ctx, batchID, nodeID, name)
	if err != nil {
		return nil, err
	}
	v := s.nodeView(n)
	return &v, nil
}

func (s *Service) DeleteNode(ctx context.Context, customerID, batchID, nodeID int) error {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return err
	}
	keys, err := s.repo.DeleteNode(ctx, batchID, nodeID)
	if err != nil {
		return err
	}
	if s.storage != nil {
		for _, k := range keys {
			_ = s.storage.Remove(ctx, k)
		}
	}
	return nil
}

func (s *Service) view(ctx context.Context, b *Batch) (*BatchView, error) {
	assignments, err := s.repo.LoadModuleAssignments(ctx, b.ID, b.CourseID)
	if err != nil {
		return nil, err
	}
	tutorCount, err := s.repo.CountTutors(ctx, b.ID)
	if err != nil {
		return nil, err
	}
	studentCount, err := s.repo.CountStudents(ctx, b.ID)
	if err != nil {
		return nil, err
	}

	modules := make([]ModuleAssignmentView, 0, len(assignments))
	for _, a := range assignments {
		mv := ModuleAssignmentView{
			CourseModuleID: a.CourseModuleID,
			ModuleTitle:    a.ModuleTitle,
			ModulePosition: a.ModulePosition,
		}
		if a.StartDate != nil {
			mv.StartDate = a.StartDate.Format(dateLayout)
		}
		if a.ExpectedEndDate != nil {
			mv.ExpectedEndDate = a.ExpectedEndDate.Format(dateLayout)
		}
		if a.TutorID != nil {
			mv.Tutor = &TutorSummaryView{
				ID:        *a.TutorID,
				FirstName: firstNonEmpty(a.TutorFirstName),
				LastName:  firstNonEmpty(a.TutorLastName),
				Email:     firstNonEmpty(a.TutorEmail),
			}
		}
		modules = append(modules, mv)
	}

	return &BatchView{
		ID: b.ID, CourseID: b.CourseID, Name: b.Name, Status: b.Status,
		PublishedAt: b.PublishedAt, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
		Modules: modules, TutorCount: tutorCount, StudentCount: studentCount,
	}, nil
}

func (s *Service) nodeView(n *DriveNode) DriveNodeView {
	v := DriveNodeView{
		ID: n.ID, ParentID: n.ParentID, Name: n.Name, Type: n.NodeType,
		ContentType: n.ContentType, SizeBytes: n.SizeBytes, CreatedAt: n.CreatedAt,
	}
	if s.storage != nil && n.ObjectKey != nil && *n.ObjectKey != "" {
		v.URL = s.storage.PublicURL(*n.ObjectKey)
	}
	return v
}

func tutorSummaryView(t TutorSummary) TutorSummaryView {
	return TutorSummaryView{ID: t.ID, FirstName: t.FirstName, LastName: t.LastName, Email: t.Email}
}

func studentSummaryView(st StudentSummary) StudentSummaryView {
	return StudentSummaryView{ID: st.ID, FirstName: st.FirstName, LastName: st.LastName, Email: st.Email, Phone: st.Phone}
}
