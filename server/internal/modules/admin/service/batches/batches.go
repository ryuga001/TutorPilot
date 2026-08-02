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

	dto "tutorpilot/internal/modules/admin/dto/batches"
	model "tutorpilot/internal/modules/admin/model/batches"
	repository "tutorpilot/internal/modules/admin/repository/batches"
	"tutorpilot/internal/modules/admin/storage"
	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/pkg/events"
	"tutorpilot/internal/pkg/httpx"
	"tutorpilot/internal/pkg/outbox"
	"tutorpilot/internal/pkg/pg"
	"tutorpilot/internal/pkg/security"
)

const dateLayout = "2006-01-02"

type Service struct {
	repo    *repository.Repository
	storage *storage.Storage
	db      pg.Beginner
	pepper  string

	signInURL     string
	stream        string
	importMaxRows int
}

type ServiceConfig struct {
	Pepper        string
	SignInURL     string
	Stream        string
	ImportMaxRows int
}

func NewService(repo *repository.Repository, store *storage.Storage, db pg.Beginner, cfg ServiceConfig) *Service {
	return &Service{
		repo:          repo,
		storage:       store,
		db:            db,
		pepper:        cfg.Pepper,
		signInURL:     cfg.SignInURL,
		stream:        cfg.Stream,
		importMaxRows: cfg.ImportMaxRows,
	}
}

func (s *Service) Create(ctx context.Context, customerID, userID, courseID int, name string) (*model.BatchView, error) {
	b, err := s.repo.CreateBatch(ctx, customerID, userID, courseID, name)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, b)
}

func (s *Service) List(ctx context.Context, customerID int, courseID *int, status, search string, p httpx.Page) (httpx.Paginated[model.BatchView], error) {
	rows, total, err := s.repo.ListBatches(ctx, customerID, courseID, status, search, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[model.BatchView]{}, err
	}
	items := make([]model.BatchView, 0, len(rows))
	for i := range rows {
		v, err := s.view(ctx, &rows[i])
		if err != nil {
			return httpx.Paginated[model.BatchView]{}, err
		}
		items = append(items, *v)
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (*model.BatchView, error) {
	b, err := s.repo.GetBatch(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, b)
}

func (s *Service) Update(ctx context.Context, customerID, id int, name string) (*model.BatchView, error) {
	b, err := s.repo.UpdateBatch(ctx, customerID, id, name)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, b)
}

func (s *Service) Delete(ctx context.Context, customerID, id int) error {
	return s.repo.DeleteBatch(ctx, customerID, id)
}

func (s *Service) SetPublished(ctx context.Context, customerID, id int, published bool) (*model.BatchView, error) {
	status := model.StatusDraft
	var at *time.Time
	if published {
		status = model.StatusPublished
		now := time.Now()
		at = &now
	}

	b, err := pg.InTx(ctx, s.db, func(q pg.Querier) (*model.Batch, error) {
		b, err := s.repo.SetStatusTx(ctx, q, customerID, id, status, at)
		if err != nil {
			return nil, err
		}
		if !published {
			return b, nil
		}
		evts, err := s.publishEvents(ctx, q, b)
		if err != nil {
			return nil, err
		}
		if s.stream != "" && len(evts) > 0 {
			if err := outbox.InsertBatch(ctx, q, s.stream, evts); err != nil {
				return nil, err
			}
		}
		return b, nil
	})
	if err != nil {
		return nil, err
	}

	return s.view(ctx, b)
}

func (s *Service) publishEvents(ctx context.Context, q pg.Querier, b *model.Batch) ([]events.Event, error) {
	courseTitle, err := s.repo.CourseTitleTx(ctx, q, b.CourseID)
	if err != nil {
		log.Printf("batches: could not load course title for batch %d: %v", b.ID, err)
		courseTitle = ""
	}

	now := time.Now()
	var out []events.Event

	assignments, err := s.repo.LoadModuleAssignmentsTx(ctx, q, b.ID, b.CourseID)
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
		evt, err := events.NewEmail(events.TypeEmailRequested, b.CustomerID, now, events.EmailRequested{
			To:           *a.TutorEmail,
			TemplateName: notification.TmplBatchTutorAssignment,
			Vars:         vars,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, evt)
	}

	students, err := s.repo.LoadAllStudentsTx(ctx, q, b.ID)
	if err != nil {
		log.Printf("batches: could not load students for batch %d: %v", b.ID, err)
		return out, nil
	}
	studentEvts, err := studentEnrolmentEvents(b, courseTitle, students, now)
	if err != nil {
		return nil, err
	}
	return append(out, studentEvts...), nil
}

func studentEnrolmentEvents(b *model.Batch, courseTitle string, students []model.StudentSummary, now time.Time) ([]events.Event, error) {
	out := make([]events.Event, 0, len(students))
	for _, st := range students {
		evt, err := events.NewEmail(events.TypeEmailRequested, b.CustomerID, now, events.EmailRequested{
			To:           st.Email,
			TemplateName: notification.TmplBatchStudentEnrollment,
			Vars: map[string]string{
				"name":        st.FirstName,
				"batch_name":  b.Name,
				"course_name": courseTitle,
			},
		})
		if err != nil {
			return nil, err
		}
		out = append(out, evt)
	}
	return out, nil
}

func firstNonEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Service) AssignTutor(ctx context.Context, customerID, batchID, courseModuleID int, req dto.AssignTutorRequest) error {
	start, err := time.Parse(dateLayout, req.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start_date, expected YYYY-MM-DD")
	}
	end, err := time.Parse(dateLayout, req.ExpectedEndDate)
	if err != nil {
		return fmt.Errorf("invalid expected_end_date, expected YYYY-MM-DD")
	}
	if end.Before(start) {
		return model.ErrInvalidDateRange
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

func (s *Service) ListTutors(ctx context.Context, customerID, batchID int) ([]model.TutorSummaryView, error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}
	rows, err := s.repo.LoadTutors(ctx, batchID)
	if err != nil {
		return nil, err
	}
	out := make([]model.TutorSummaryView, 0, len(rows))
	for _, t := range rows {
		out = append(out, tutorSummaryView(t))
	}
	return out, nil
}

func (s *Service) ListStudents(ctx context.Context, customerID, batchID int, p httpx.Page) (httpx.Paginated[model.StudentSummaryView], error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return httpx.Paginated[model.StudentSummaryView]{}, err
	}
	rows, total, err := s.repo.LoadStudents(ctx, batchID, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[model.StudentSummaryView]{}, err
	}
	items := make([]model.StudentSummaryView, 0, len(rows))
	for _, st := range rows {
		items = append(items, studentSummaryView(st))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) EnrollStudents(ctx context.Context, customerID, batchID int, studentIDs []int) (*model.EnrollResult, error) {
	b, err := s.repo.GetBatch(ctx, customerID, batchID)
	if err != nil {
		return nil, err
	}

	return pg.InTx(ctx, s.db, func(q pg.Querier) (*model.EnrollResult, error) {
		enrolled, newlyEnrolled, notFound, err := s.repo.EnrollStudentIDs(ctx, q, customerID, batchID, studentIDs)
		if err != nil {
			return nil, err
		}

		if b.Status == model.StatusPublished && len(newlyEnrolled) > 0 {
			evts, err := s.enrolmentEvents(ctx, q, b, newlyEnrolled)
			if err != nil {
				return nil, err
			}
			if s.stream != "" && len(evts) > 0 {
				if err := outbox.InsertBatch(ctx, q, s.stream, evts); err != nil {
					return nil, err
				}
			}
		}

		return &model.EnrollResult{Enrolled: enrolled, NotFound: notFound}, nil
	})
}

func (s *Service) enrolmentEvents(ctx context.Context, q pg.Querier, b *model.Batch, studentIDs []int) ([]events.Event, error) {
	courseTitle, err := s.repo.CourseTitleTx(ctx, q, b.CourseID)
	if err != nil {
		log.Printf("batches: could not load course title for batch %d: %v", b.ID, err)
		courseTitle = ""
	}

	students, err := s.repo.LoadStudentsByIDs(ctx, q, studentIDs)
	if err != nil {
		return nil, err
	}
	return studentEnrolmentEvents(b, courseTitle, students, time.Now())
}

func (s *Service) RemoveStudent(ctx context.Context, customerID, batchID, studentID int) error {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return err
	}
	return s.repo.RemoveStudent(ctx, batchID, studentID)
}

func (s *Service) ImportStudentsCSV(ctx context.Context, customerID, batchID int, r io.Reader) (*model.ImportResult, error) {
	batch, err := s.repo.GetBatch(ctx, customerID, batchID)
	if err != nil {
		return nil, err
	}

	courseTitle, err := s.repo.CourseTitle(ctx, batch.CourseID)
	if err != nil {
		log.Printf("batches: could not load course title for batch %d: %v", batch.ID, err)
		courseTitle = ""
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

	valid := make([]repository.ImportRow, 0)
	skipped := make([]model.SkippedRow, 0)
	rowNum := 1

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		rowNum++
		if err != nil {
			skipped = append(skipped, model.SkippedRow{Row: rowNum, Reason: "could not parse row"})
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

		if len(valid) >= s.importMaxRows {
			skipped = append(skipped, model.SkippedRow{
				Row:    rowNum,
				Reason: fmt.Sprintf("import is limited to %d rows per upload", s.importMaxRows),
			})
			continue
		}
		if first == "" || last == "" {
			skipped = append(skipped, model.SkippedRow{Row: rowNum, Reason: "missing first or last name"})
			continue
		}
		if _, err := mail.ParseAddress(email); err != nil {
			skipped = append(skipped, model.SkippedRow{Row: rowNum, Reason: "invalid email"})
			continue
		}

		tempPassword, err := security.GenerateTempPassword()
		if err != nil {
			return nil, err
		}
		salt, err := security.NewSalt()
		if err != nil {
			return nil, err
		}
		hash, err := security.HashPassword(tempPassword, salt, s.pepper)
		if err != nil {
			return nil, err
		}

		invite, err := events.NewEmail(events.TypeEmailRequested, customerID, time.Now(), events.EmailRequested{
			To:           email,
			TemplateName: notification.TmplMemberInvite,
			Vars: notification.InviteVars(notification.MemberInvite{
				Name:         first,
				Role:         "student",
				Email:        email,
				TempPassword: tempPassword,
				SignInURL:    s.signInURL,
			}),
		})
		if err != nil {
			return nil, err
		}

		enrolment, err := events.NewEmail(events.TypeEmailRequested, customerID, time.Now(), events.EmailRequested{
			To:           email,
			TemplateName: notification.TmplBatchStudentEnrollment,
			Vars: map[string]string{
				"name":        first,
				"batch_name":  batch.Name,
				"course_name": courseTitle,
			},
		})
		if err != nil {
			return nil, err
		}

		valid = append(valid, repository.ImportRow{
			StudentRow: repository.StudentRow{FirstName: first, LastName: last, Email: email, Phone: phone},
			Row:                   rowNum,
			PasswordHash:          hash,
			PasswordSalt:          salt,
			InviteEvent:           invite,
			EnrolmentEvent:        enrolment,
		})
	}

	imported := 0
	if len(valid) > 0 {
		outcomes, repoSkipped, err := s.repo.ImportStudents(ctx, repository.ImportParams{
			CustomerID: customerID,
			BatchID:    batchID,
			Stream:     s.stream,

			NotifyEnrolment: batch.Status == model.StatusPublished,
		}, valid)
		if err != nil {
			return nil, err
		}
		skipped = append(skipped, repoSkipped...)
		imported = len(outcomes)
	}
	if imported == 0 && len(skipped) == 0 {
		return nil, model.ErrEmptyImport
	}

	return &model.ImportResult{Imported: imported, Skipped: skipped}, nil
}

func (s *Service) ListDrive(ctx context.Context, customerID, batchID int, parentID *int) ([]model.DriveNodeView, error) {
	if _, err := s.repo.GetBatch(ctx, customerID, batchID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListChildren(ctx, batchID, parentID)
	if err != nil {
		return nil, err
	}
	out := make([]model.DriveNodeView, 0, len(rows))
	for i := range rows {
		out = append(out, s.nodeView(&rows[i]))
	}
	return out, nil
}

func (s *Service) CreateFolder(ctx context.Context, customerID, userID, batchID int, name string, parentID *int) (*model.DriveNodeView, error) {
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

func (s *Service) UploadFile(ctx context.Context, customerID, userID, batchID int, parentID *int, fh *multipart.FileHeader) (*model.DriveNodeView, error) {
	if s.storage == nil {
		return nil, model.ErrStorageUnavailable
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

func (s *Service) RenameNode(ctx context.Context, customerID, batchID, nodeID int, name string) (*model.DriveNodeView, error) {
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

func (s *Service) view(ctx context.Context, b *model.Batch) (*model.BatchView, error) {
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

	modules := make([]model.ModuleAssignmentView, 0, len(assignments))
	for _, a := range assignments {
		mv := model.ModuleAssignmentView{
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
			mv.Tutor = &model.TutorSummaryView{
				ID:        *a.TutorID,
				FirstName: firstNonEmpty(a.TutorFirstName),
				LastName:  firstNonEmpty(a.TutorLastName),
				Email:     firstNonEmpty(a.TutorEmail),
			}
		}
		modules = append(modules, mv)
	}

	return &model.BatchView{
		ID: b.ID, CourseID: b.CourseID, Name: b.Name, Status: b.Status,
		PublishedAt: b.PublishedAt, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
		Modules: modules, TutorCount: tutorCount, StudentCount: studentCount,
	}, nil
}

func (s *Service) nodeView(n *model.DriveNode) model.DriveNodeView {
	v := model.DriveNodeView{
		ID: n.ID, ParentID: n.ParentID, Name: n.Name, Type: n.NodeType,
		ContentType: n.ContentType, SizeBytes: n.SizeBytes, IsSystem: n.IsSystem,
		CreatedAt: n.CreatedAt,
	}
	if s.storage != nil && n.ObjectKey != nil && *n.ObjectKey != "" {
		v.URL = s.storage.PublicURL(*n.ObjectKey)
	}
	return v
}

func tutorSummaryView(t model.TutorSummary) model.TutorSummaryView {
	return model.TutorSummaryView{ID: t.ID, FirstName: t.FirstName, LastName: t.LastName, Email: t.Email}
}

func studentSummaryView(st model.StudentSummary) model.StudentSummaryView {
	return model.StudentSummaryView{ID: st.ID, FirstName: st.FirstName, LastName: st.LastName, Email: st.Email, Phone: st.Phone}
}
