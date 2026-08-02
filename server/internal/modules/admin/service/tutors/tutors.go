package tutors

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"tutorpilot/internal/modules/admin/address"
	dto "tutorpilot/internal/modules/admin/dto/tutors"
	model "tutorpilot/internal/modules/admin/model/tutors"
	repository "tutorpilot/internal/modules/admin/repository/tutors"
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

func (s *Service) Create(ctx context.Context, customerID, userID int, req dto.CreateTutorRequest) (*model.CreatedTutor, error) {
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
			Role:         "tutor",
			Email:        req.Email,
			TempPassword: tempPassword,
			SignInURL:    s.signInURL,
		}),
	})
	if err != nil {
		return nil, err
	}

	t, err := s.repo.Create(ctx, customerID, userID, repository.CreateInput{
		CreateTutorRequest: req,
		PasswordHash:       hash,
		PasswordSalt:       salt,
		InviteEvent:        invite,
		InviteStream:       s.stream,
	})
	if err != nil {
		return nil, err
	}

	return &model.CreatedTutor{TutorView: *s.view(ctx, customerID, t), TempPassword: tempPassword}, nil
}

func (s *Service) List(ctx context.Context, customerID int, search string, p httpx.Page) (httpx.Paginated[model.TutorView], error) {
	rows, total, err := s.repo.List(ctx, customerID, search, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[model.TutorView]{}, err
	}
	items := make([]model.TutorView, 0, len(rows))
	for i := range rows {
		items = append(items, *s.view(ctx, customerID, &rows[i]))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (*model.TutorView, error) {
	t, err := s.repo.Get(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, customerID, t), nil
}

func (s *Service) Update(ctx context.Context, customerID, id int, req dto.UpdateTutorRequest) (*model.TutorView, error) {
	t, err := s.repo.Update(ctx, customerID, id, req)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, customerID, t), nil
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

func (s *Service) UploadProfileImage(ctx context.Context, customerID, id int, fh *multipart.FileHeader) (*model.TutorView, error) {
	if s.storage == nil {
		return nil, model.ErrStorageUnavailable
	}
	current, err := s.repo.Get(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("customer_%d/tutors/%d/profile", customerID, id)
	key, err := s.storage.UploadFile(ctx, prefix, fh)
	if err != nil {
		return nil, err
	}
	t, err := s.repo.SetProfileImage(ctx, customerID, id, key)
	if err != nil {
		_ = s.storage.Remove(ctx, key)
		return nil, err
	}
	if current.ProfileImageKey != nil && *current.ProfileImageKey != "" {
		_ = s.storage.Remove(ctx, *current.ProfileImageKey)
	}
	return s.view(ctx, customerID, t), nil
}

func (s *Service) view(ctx context.Context, customerID int, t *model.Tutor) *model.TutorView {
	v := &model.TutorView{
		ID: t.ID, FirstName: t.FirstName, LastName: t.LastName, Email: t.Email,
		PhoneNo: t.PhoneNo, Designation: t.Designation,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
	if s.storage != nil && t.ProfileImageKey != nil && *t.ProfileImageKey != "" {
		v.ProfileImageURL = s.storage.PublicURL(*t.ProfileImageKey)
	}
	if t.AddressID != nil {
		if a, err := address.Get(ctx, s.db, customerID, *t.AddressID); err == nil {
			v.Address = a.View()
		}
	}
	return v
}
