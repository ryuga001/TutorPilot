package lecture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	lkprotocol "github.com/livekit/protocol/livekit"

	"tutorpilot/internal/livekit"
	"tutorpilot/internal/pkg/httpx"
	"tutorpilot/internal/pkg/scope"
	"tutorpilot/internal/pkg/storage"
)

var (
	ErrLiveKitUnavailable = errors.New("live video is not configured")
	ErrNotLive            = errors.New("this lecture is not live")
	ErrNoRecording        = errors.New("this lecture has no recording")
)

type LiveKitClient interface {
	IsConfigured() bool
	CreateRoom(ctx context.Context, roomName, metadata string) error
	DeleteRoom(ctx context.Context, roomName string) error
	StartRoomCompositeEgress(ctx context.Context, roomName, objectKey string, s *storage.Storage) (*lkprotocol.EgressInfo, error)
	StopEgress(ctx context.Context, egressID string) (*lkprotocol.EgressInfo, error)
	GenerateToken(identity, name, metadata string, ttl time.Duration, grant livekit.VideoGrant) (string, error)
}

// PrivilegeChecker reports whether the caller holds a privilege. Join uses it to
// decide publish rights, so the grants in the token follow the same role
// definitions as everything else instead of a hardcoded list of role names.
type PrivilegeChecker func(ctx context.Context, userID, privilege string) (bool, error)

type Service struct {
	repo       *Repository
	liveClient LiveKitClient
	storage    *storage.Storage
	hasPriv    PrivilegeChecker
	joinTTL    time.Duration

	// Supplied after construction via SetDrive: recordings are filed into the batch
	// drive, which a different module owns.
	drive     DriveWriter
	objectURL ObjectURLFunc
}

func NewService(
	repo *Repository,
	lc LiveKitClient,
	store *storage.Storage,
	hasPriv PrivilegeChecker,
	joinTTL time.Duration,
) *Service {
	if joinTTL <= 0 {
		joinTTL = 2 * time.Hour
	}
	return &Service{repo: repo, liveClient: lc, storage: store, hasPriv: hasPriv, joinTTL: joinTTL}
}

func (s *Service) liveKitReady() bool {
	return s.liveClient != nil && s.liveClient.IsConfigured()
}

// Create records the lecture and reserves a room name. It does not create the room:
// LiveKit reaps an empty room after its timeout, so a room made now would be gone
// before a lecture scheduled for tomorrow began, with nothing to recreate it.
func (s *Service) Create(ctx context.Context, sc scope.Scope, userID int, req CreateLectureRequest) (*LectureView, error) {
	roomName := "lecture-" + uuid.New().String()

	l, err := s.repo.Create(ctx, sc, userID, roomName, req)
	if err != nil {
		return nil, err
	}
	v := view(l)
	return &v, nil
}

func (s *Service) List(
	ctx context.Context,
	sc scope.Scope,
	f ListLectureFilter,
	p httpx.Page,
) (httpx.Paginated[LectureView], error) {
	rows, total, err := s.repo.List(ctx, sc, f, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[LectureView]{}, err
	}
	items := make([]LectureView, 0, len(rows))
	for i := range rows {
		items = append(items, view(&rows[i]))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, sc scope.Scope, userID int, id int64) (*LectureView, error) {
	l, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	v := view(l)
	v.CanPublish = s.canPublish(ctx, userID)
	return &v, nil
}

func (s *Service) Update(ctx context.Context, sc scope.Scope, id int64, req UpdateLectureRequest) (*LectureView, error) {
	l, err := s.repo.Update(ctx, sc, id, req)
	if err != nil {
		return nil, err
	}
	v := view(l)
	return &v, nil
}

func (s *Service) Delete(ctx context.Context, sc scope.Scope, id int64) error {
	l, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, sc, id); err != nil {
		return err
	}
	if s.liveKitReady() && l.RoomName != nil && *l.RoomName != "" {
		if err := s.liveClient.DeleteRoom(ctx, *l.RoomName); err != nil {
			log.Printf("lecture: could not delete room %s: %v", *l.RoomName, err)
		}
	}
	return nil
}

// Start opens the room and begins recording. It is idempotent — a lecture already
// live is returned as-is rather than starting a second egress job.
func (s *Service) Start(ctx context.Context, sc scope.Scope, id int64) (*LectureView, error) {
	current, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if current.Status == StatusLive {
		v := view(current)
		return &v, nil
	}
	if current.RoomName == nil || *current.RoomName == "" {
		return nil, errors.New("lecture has no room reserved")
	}

	// Whoever wins this update owns the start; a concurrent caller gets
	// ErrInvalidTransition and launches nothing.
	l, err := s.repo.Transition(ctx, sc, id, StatusScheduled, StatusLive)
	if err != nil {
		return nil, err
	}

	if !s.liveKitReady() {
		// Usable without live video configured: the lecture is tracked, there is
		// just nothing to join.
		v := view(l)
		return &v, nil
	}

	metadata, _ := json.Marshal(map[string]any{"lecture_id": l.ID, "customer_id": l.CustomerID})
	if err := s.liveClient.CreateRoom(ctx, *l.RoomName, string(metadata)); err != nil {
		log.Printf("lecture %d: could not create room: %v", l.ID, err)
	}

	if l.RecordingEnabled && s.storage != nil {
		s.startRecording(ctx, l)
		// Re-read so the response carries the recording status just written.
		if refreshed, err := s.repo.Get(ctx, sc, id); err == nil {
			l = refreshed
		}
	}

	v := view(l)
	return &v, nil
}

// startRecording launches egress and persists the real egress id along with the key
// the file will land at. A failure here is logged and surfaced as a failed recording
// status, never as a failed request: the class itself is running.
func (s *Service) startRecording(ctx context.Context, l *Lecture) {
	if err := s.repo.SetRecordingStatus(ctx, nil, l.ID, RecordingStarting); err != nil {
		log.Printf("lecture %d: could not mark recording starting: %v", l.ID, err)
	}

	// Written straight into the batch's own drive prefix, so finishing the recording
	// is a database insert rather than a copy.
	objectKey := fmt.Sprintf("customer_%d/batches/%d/drive/lectures/%d/recording-%s.mp4",
		l.CustomerID, l.BatchID, l.ID, uuid.New().String())

	info, err := s.liveClient.StartRoomCompositeEgress(ctx, *l.RoomName, objectKey, s.storage)
	if err != nil || info == nil || info.EgressId == "" {
		log.Printf("lecture %d: could not start recording: %v", l.ID, err)
		if err := s.repo.SetRecordingStatus(ctx, nil, l.ID, RecordingFailed); err != nil {
			log.Printf("lecture %d: could not mark recording failed: %v", l.ID, err)
		}
		return
	}

	if err := s.repo.SetRecordingTarget(ctx, l.CustomerID, l.ID, info.EgressId, objectKey); err != nil {
		log.Printf("lecture %d: could not store recording target: %v", l.ID, err)
	}
}

// End closes the lecture. Recording moves to processing rather than ready: egress
// finalises asynchronously and the egress_ended webhook reports the finished file.
func (s *Service) End(ctx context.Context, sc scope.Scope, id int64) (*LectureView, error) {
	current, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if current.Status == StatusEnded {
		v := view(current)
		return &v, nil
	}

	l, err := s.repo.Transition(ctx, sc, id, StatusLive, StatusEnded)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CloseOpenAttendance(ctx, id); err != nil {
		log.Printf("lecture %d: could not close attendance: %v", id, err)
	}

	if s.liveKitReady() {
		if l.EgressID != nil && *l.EgressID != "" {
			if _, err := s.liveClient.StopEgress(ctx, *l.EgressID); err != nil {
				log.Printf("lecture %d: could not stop recording: %v", id, err)
				_ = s.repo.SetRecordingStatus(ctx, nil, id, RecordingFailed)
			} else {
				_ = s.repo.SetRecordingStatus(ctx, nil, id, RecordingProcessing)
			}
		}
		if l.RoomName != nil && *l.RoomName != "" {
			if err := s.liveClient.DeleteRoom(ctx, *l.RoomName); err != nil {
				log.Printf("lecture %d: could not delete room: %v", id, err)
			}
		}
	}

	if refreshed, err := s.repo.Get(ctx, sc, id); err == nil {
		l = refreshed
	}
	v := view(l)
	return &v, nil
}

// Cancel calls off a lecture that has not started.
func (s *Service) Cancel(ctx context.Context, sc scope.Scope, id int64) (*LectureView, error) {
	l, err := s.repo.Transition(ctx, sc, id, StatusScheduled, StatusCancelled)
	if err != nil {
		return nil, err
	}
	if s.liveKitReady() && l.RoomName != nil && *l.RoomName != "" {
		// The room may never have been created; deleting a missing one is harmless.
		_ = s.liveClient.DeleteRoom(ctx, *l.RoomName)
	}
	v := view(l)
	return &v, nil
}

// Join issues a LiveKit credential.
//
// Authorization is the scope check that loaded the lecture plus the lecture.join
// privilege on the route. Publish rights come from lecture.publish, so a student
// subscribes and sends data — chat, raise-hand — but cannot open a camera.
func (s *Service) Join(ctx context.Context, sc scope.Scope, userID int, id int64) (*JoinView, error) {
	l, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if l.RoomName == nil || *l.RoomName == "" {
		return nil, ErrLiveKitUnavailable
	}
	if l.Status != StatusLive {
		return nil, ErrNotLive
	}
	if !s.liveKitReady() {
		return nil, ErrLiveKitUnavailable
	}

	canPublish := s.canPublish(ctx, userID)
	displayName := s.repo.DisplayNameForUser(ctx, sc.CustomerID, userID)

	// A stable opaque identity, so participants are not shown each other's email.
	// Whether this user is a tutor, student, or admin is derived server-side from
	// their id when attendance is recorded, so it does not need to travel here.
	identity := "u:" + strconv.Itoa(userID)
	metadata, _ := json.Marshal(map[string]any{
		"user_id": userID,
		"name":    displayName,
	})

	token, err := s.liveClient.GenerateToken(identity, displayName, string(metadata), s.joinTTL, livekit.VideoGrant{
		RoomJoin:       true,
		Room:           *l.RoomName,
		CanPublish:     canPublish,
		CanSubscribe:   true,
		CanPublishData: true,
	})
	if err != nil {
		return nil, err
	}
	return &JoinView{
		Token:      token,
		RoomName:   *l.RoomName,
		Identity:   identity,
		CanPublish: canPublish,
	}, nil
}

func (s *Service) canPublish(ctx context.Context, userID int) bool {
	if s.hasPriv == nil {
		return false
	}
	ok, err := s.hasPriv(ctx, strconv.Itoa(userID), "lecture.publish")
	if err != nil {
		log.Printf("lecture: could not check publish privilege for user %d: %v", userID, err)
		return false
	}
	return ok
}

func (s *Service) Attendance(ctx context.Context, sc scope.Scope, id int64) ([]AttendanceView, error) {
	if _, err := s.repo.Get(ctx, sc, id); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListAttendance(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]AttendanceView, 0, len(rows))
	for i := range rows {
		out = append(out, attendanceView(&rows[i]))
	}
	return out, nil
}

// RecordingURL returns where the finished recording can be fetched.
func (s *Service) RecordingURL(ctx context.Context, sc scope.Scope, id int64) (string, error) {
	l, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return "", err
	}
	if l.RecordingURL == nil || *l.RecordingURL == "" {
		return "", ErrNoRecording
	}
	return *l.RecordingURL, nil
}
