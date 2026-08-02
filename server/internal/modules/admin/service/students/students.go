package students

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"tutorpilot/internal/modules/admin/address"
	dto "tutorpilot/internal/modules/admin/dto/students"
	model "tutorpilot/internal/modules/admin/model/students"
	repository "tutorpilot/internal/modules/admin/repository/students"
	"tutorpilot/internal/modules/admin/storage"
	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/pkg/events"
	"tutorpilot/internal/pkg/httpx"
	"tutorpilot/internal/pkg/pg"
	"tutorpilot/internal/pkg/security"
)

type Service struct {
	repo    *repository.Repository
	storage *storage.Storage
	db      pg.Querier
	pepper  string

	signInURL string
	stream    string
}

func NewService(repo *repository.Repository, store *storage.Storage, db pg.Querier, pepper, signInURL, stream string) *Service {
	return &Service{repo: repo, storage: store, db: db, pepper: pepper, signInURL: signInURL, stream: stream}
}

func (s *Service) Create(ctx context.Context, customerID, userID int, req dto.CreateStudentRequest) (*model.CreatedStudent, error) {
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
		To:           req.Email,
		TemplateName: notification.TmplMemberInvite,
		Vars: notification.InviteVars(notification.MemberInvite{
			Name:         req.FirstName,
			Role:         "student",
			Email:        req.Email,
			TempPassword: tempPassword,
			SignInURL:    s.signInURL,
		}),
	})
	if err != nil {
		return nil, err
	}

	st, err := s.repo.Create(ctx, customerID, userID, repository.CreateInput{
		CreateStudentRequest: req,
		PasswordHash:         hash,
		PasswordSalt:         salt,
		InviteEvent:          invite,
		InviteStream:         s.stream,
	})
	if err != nil {
		return nil, err
	}

	return &model.CreatedStudent{StudentView: *s.view(ctx, customerID, st), TempPassword: tempPassword}, nil
}

func (s *Service) List(ctx context.Context, customerID int, search string, p httpx.Page) (httpx.Paginated[model.StudentView], error) {
	rows, total, err := s.repo.List(ctx, customerID, search, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[model.StudentView]{}, err
	}
	items := make([]model.StudentView, 0, len(rows))
	for i := range rows {
		items = append(items, *s.view(ctx, customerID, &rows[i]))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (*model.StudentView, error) {
	st, err := s.repo.Get(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, customerID, st), nil
}

func (s *Service) Update(ctx context.Context, customerID, id int, req dto.UpdateStudentRequest) (*model.StudentView, error) {
	st, err := s.repo.Update(ctx, customerID, id, req)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, customerID, st), nil
}

func (s *Service) Delete(ctx context.Context, customerID, id int) error {
	oldKey, err := s.repo.Delete(ctx, customerID, id)
	if err != nil {
		return err
	}
	if s.storage != nil && oldKey != nil && *oldKey != "" {
		_ = s.storage.Remove(ctx, *oldKey)
	}
	return nil
}

func (s *Service) UploadProfileImage(ctx context.Context, customerID, id int, fh *multipart.FileHeader) (*model.StudentView, error) {
	if s.storage == nil {
		return nil, model.ErrStorageUnavailable
	}
	current, err := s.repo.Get(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("customer_%d/students/%d/profile", customerID, id)
	key, err := s.storage.UploadFile(ctx, prefix, fh)
	if err != nil {
		return nil, err
	}
	st, err := s.repo.SetProfileImage(ctx, customerID, id, key)
	if err != nil {
		_ = s.storage.Remove(ctx, key)
		return nil, err
	}
	if current.ProfileImageKey != nil && *current.ProfileImageKey != "" {
		_ = s.storage.Remove(ctx, *current.ProfileImageKey)
	}
	return s.view(ctx, customerID, st), nil
}

func (s *Service) view(ctx context.Context, customerID int, st *model.Student) *model.StudentView {
	v := &model.StudentView{
		ID: st.ID, FirstName: st.FirstName, LastName: st.LastName, Email: st.Email,
		PhoneNo: st.PhoneNo, CreatedAt: st.CreatedAt, UpdatedAt: st.UpdatedAt,
	}
	if s.storage != nil && st.ProfileImageKey != nil && *st.ProfileImageKey != "" {
		v.ProfileImageURL = s.storage.PublicURL(*st.ProfileImageKey)
	}
	if st.AddressID != nil {
		if a, err := address.Get(ctx, s.db, customerID, *st.AddressID); err == nil {
			v.Address = a.View()
		}
	}
	return v
}
