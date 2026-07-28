package tutors

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"

	"tutorpilot/internal/pkg/address"
	"tutorpilot/internal/pkg/httpx"
	"tutorpilot/internal/pkg/mailer"
	"tutorpilot/internal/pkg/pg"
	"tutorpilot/internal/pkg/security"
	"tutorpilot/internal/pkg/storage"
)

var ErrStorageUnavailable = errors.New("file storage is not configured")

type Service struct {
	repo    *Repository
	storage *storage.Storage
	db      pg.Querier
	mail    *mailer.Mailer
	pepper  string
}

func NewService(repo *Repository, store *storage.Storage, db pg.Querier, mail *mailer.Mailer, pepper string) *Service {
	return &Service{repo: repo, storage: store, db: db, mail: mail, pepper: pepper}
}

// Create adds a tutor. A tutor is a login first: this generates a random
// temporary password, hashes it the same way every other password is hashed,
// and creates the dashboard_users + tutors rows in one transaction. The
// temporary password is returned once, in CreatedTutor, and never again — an
// admin who loses it has to use ResetPassword-equivalent tooling (not yet
// exposed here) rather than read it back.
func (s *Service) Create(ctx context.Context, customerID, userID int, req CreateTutorRequest) (*CreatedTutor, error) {
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

	t, err := s.repo.Create(ctx, customerID, userID, CreateInput{
		CreateTutorRequest: req,
		PasswordHash:       hash,
		PasswordSalt:       salt,
	})
	if err != nil {
		return nil, err
	}

	s.sendInvite(t.Email, t.FirstName, tempPassword)

	return &CreatedTutor{TutorView: *s.view(ctx, customerID, t), TempPassword: tempPassword}, nil
}

// sendInvite emails the temporary password directly. A failure here is logged,
// not returned: the login already exists and works, a bounced email must not
// undo that — the admin can still hand the password over another way (it is
// also in the Create response).
func (s *Service) sendInvite(toEmail, name, tempPassword string) {
	if s.mail == nil {
		return
	}
	subject := "Your TutorPilot tutor account"
	body := fmt.Sprintf(
		"<p>Hi %s,</p><p>An account has been created for you on TutorPilot.</p>"+
			"<p>Email: <strong>%s</strong><br>Temporary password: <strong>%s</strong></p>"+
			"<p>Sign in and change your password when convenient.</p>",
		name, toEmail, tempPassword)
	if err := s.mail.Send(toEmail, subject, body); err != nil {
		log.Printf("tutors: could not email invite to %s: %v", toEmail, err)
	}
}

func (s *Service) List(ctx context.Context, customerID int, search string, p httpx.Page) (httpx.Paginated[TutorView], error) {
	rows, total, err := s.repo.List(ctx, customerID, search, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[TutorView]{}, err
	}
	items := make([]TutorView, 0, len(rows))
	for i := range rows {
		items = append(items, *s.view(ctx, customerID, &rows[i]))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (*TutorView, error) {
	t, err := s.repo.Get(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, customerID, t), nil
}

func (s *Service) Update(ctx context.Context, customerID, id int, req UpdateTutorRequest) (*TutorView, error) {
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

func (s *Service) UploadProfileImage(ctx context.Context, customerID, id int, fh *multipart.FileHeader) (*TutorView, error) {
	if s.storage == nil {
		return nil, ErrStorageUnavailable
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

func (s *Service) view(ctx context.Context, customerID int, t *Tutor) *TutorView {
	v := &TutorView{
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
