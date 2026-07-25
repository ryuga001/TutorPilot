package students

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"

	"tutorpilot/internal/pkg/address"
	"tutorpilot/internal/pkg/httpx"
	"tutorpilot/internal/pkg/pg"
	"tutorpilot/internal/pkg/storage"
)

var ErrStorageUnavailable = errors.New("file storage is not configured")

type Service struct {
	repo    *Repository
	storage *storage.Storage
	db      pg.Querier
}

func NewService(repo *Repository, store *storage.Storage, db pg.Querier) *Service {
	return &Service{repo: repo, storage: store, db: db}
}

func (s *Service) Create(ctx context.Context, customerID, userID int, req CreateStudentRequest) (*StudentView, error) {
	st, err := s.repo.Create(ctx, customerID, userID, req)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, customerID, st), nil
}

func (s *Service) List(ctx context.Context, customerID int, search string, p httpx.Page) (httpx.Paginated[StudentView], error) {
	rows, total, err := s.repo.List(ctx, customerID, search, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[StudentView]{}, err
	}
	items := make([]StudentView, 0, len(rows))
	for i := range rows {
		items = append(items, *s.view(ctx, customerID, &rows[i]))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (*StudentView, error) {
	st, err := s.repo.Get(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, customerID, st), nil
}

func (s *Service) Update(ctx context.Context, customerID, id int, req UpdateStudentRequest) (*StudentView, error) {
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

func (s *Service) UploadProfileImage(ctx context.Context, customerID, id int, fh *multipart.FileHeader) (*StudentView, error) {
	if s.storage == nil {
		return nil, ErrStorageUnavailable
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

func (s *Service) view(ctx context.Context, customerID int, st *Student) *StudentView {
	v := &StudentView{
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
