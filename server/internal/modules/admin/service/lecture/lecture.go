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

	dto "tutorpilot/internal/modules/admin/dto/lecture"
	"tutorpilot/internal/modules/admin/livekit"
	model "tutorpilot/internal/modules/admin/model/lecture"
	repository "tutorpilot/internal/modules/admin/repository/lecture"
	"tutorpilot/internal/modules/admin/scope"
	"tutorpilot/internal/modules/admin/storage"
	"tutorpilot/internal/pkg/httpx"
)

type LiveKitClient interface {
	IsConfigured() bool
	CreateRoom(ctx context.Context, roomName, metadata string) error
	DeleteRoom(ctx context.Context, roomName string) error
	StartRoomCompositeEgress(ctx context.Context, roomName, objectKey string, s *storage.Storage) (*lkprotocol.EgressInfo, error)
	StopEgress(ctx context.Context, egressID string) (*lkprotocol.EgressInfo, error)
	GenerateToken(identity, name, metadata string, ttl time.Duration, grant livekit.VideoGrant) (string, error)
}

type PrivilegeChecker func(ctx context.Context, userID, privilege string) (bool, error)

type Service struct {
	repo       *repository.Repository
	liveClient LiveKitClient
	storage    *storage.Storage
	hasPriv    PrivilegeChecker
	joinTTL    time.Duration

	drive     DriveWriter
	objectURL ObjectURLFunc
}

func NewService(
	repo *repository.Repository,
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

func (s *Service) Create(ctx context.Context, sc scope.Scope, userID int, req dto.CreateLectureRequest) (*dto.LectureView, error) {
	roomName := "lecture-" + uuid.New().String()

	l, err := s.repo.Create(ctx, sc, userID, roomName, req)
	if err != nil {
		return nil, err
	}
	v := dto.View(l)
	return &v, nil
}

func (s *Service) List(
	ctx context.Context,
	sc scope.Scope,
	f dto.ListLectureFilter,
	p httpx.Page,
) (httpx.Paginated[dto.LectureView], error) {
	rows, total, err := s.repo.List(ctx, sc, f, p.Limit(), p.Offset())
	if err != nil {
		return httpx.Paginated[dto.LectureView]{}, err
	}
	items := make([]dto.LectureView, 0, len(rows))
	for i := range rows {
		items = append(items, dto.View(&rows[i]))
	}
	return httpx.NewPaginated(items, total, p), nil
}

func (s *Service) Get(ctx context.Context, sc scope.Scope, userID int, id int64) (*dto.LectureView, error) {
	l, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	v := dto.View(l)
	v.CanPublish = s.canPublish(ctx, userID)
	return &v, nil
}

func (s *Service) Update(ctx context.Context, sc scope.Scope, id int64, req dto.UpdateLectureRequest) (*dto.LectureView, error) {
	l, err := s.repo.Update(ctx, sc, id, req)
	if err != nil {
		return nil, err
	}
	v := dto.View(l)
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

func (s *Service) Start(ctx context.Context, sc scope.Scope, id int64) (*dto.LectureView, error) {
	current, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if current.Status == model.StatusLive {
		v := dto.View(current)
		return &v, nil
	}
	if current.RoomName == nil || *current.RoomName == "" {
		return nil, errors.New("lecture has no room reserved")
	}

	l, err := s.repo.Transition(ctx, sc, id, model.StatusScheduled, model.StatusLive)
	if err != nil {
		return nil, err
	}

	if !s.liveKitReady() {
		v := dto.View(l)
		return &v, nil
	}

	metadata, _ := json.Marshal(map[string]any{"lecture_id": l.ID, "customer_id": l.CustomerID})
	if err := s.liveClient.CreateRoom(ctx, *l.RoomName, string(metadata)); err != nil {
		log.Printf("lecture %d: could not create room: %v", l.ID, err)
	}

	if l.RecordingEnabled && s.storage != nil {
		s.startRecording(ctx, l)

		if refreshed, err := s.repo.Get(ctx, sc, id); err == nil {
			l = refreshed
		}
	}

	v := dto.View(l)
	return &v, nil
}

func (s *Service) startRecording(ctx context.Context, l *model.Lecture) {
	if err := s.repo.SetRecordingStatus(ctx, nil, l.ID, model.RecordingStarting); err != nil {
		log.Printf("lecture %d: could not mark recording starting: %v", l.ID, err)
	}

	objectKey := fmt.Sprintf("customer_%d/batches/%d/drive/lectures/%d/recording-%s.mp4",
		l.CustomerID, l.BatchID, l.ID, uuid.New().String())

	info, err := s.liveClient.StartRoomCompositeEgress(ctx, *l.RoomName, objectKey, s.storage)
	if err != nil || info == nil || info.EgressId == "" {
		log.Printf("lecture %d: could not start recording: %v", l.ID, err)
		if err := s.repo.SetRecordingStatus(ctx, nil, l.ID, model.RecordingFailed); err != nil {
			log.Printf("lecture %d: could not mark recording failed: %v", l.ID, err)
		}
		return
	}

	if err := s.repo.SetRecordingTarget(ctx, l.CustomerID, l.ID, info.EgressId, objectKey); err != nil {
		log.Printf("lecture %d: could not store recording target: %v", l.ID, err)
	}
}

func (s *Service) End(ctx context.Context, sc scope.Scope, id int64) (*dto.LectureView, error) {
	current, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if current.Status == model.StatusEnded {
		v := dto.View(current)
		return &v, nil
	}

	l, err := s.repo.Transition(ctx, sc, id, model.StatusLive, model.StatusEnded)
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
				_ = s.repo.SetRecordingStatus(ctx, nil, id, model.RecordingFailed)
			} else {
				_ = s.repo.SetRecordingStatus(ctx, nil, id, model.RecordingProcessing)
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
	v := dto.View(l)
	return &v, nil
}

func (s *Service) Cancel(ctx context.Context, sc scope.Scope, id int64) (*dto.LectureView, error) {
	l, err := s.repo.Transition(ctx, sc, id, model.StatusScheduled, model.StatusCancelled)
	if err != nil {
		return nil, err
	}
	if s.liveKitReady() && l.RoomName != nil && *l.RoomName != "" {
		_ = s.liveClient.DeleteRoom(ctx, *l.RoomName)
	}
	v := dto.View(l)
	return &v, nil
}

func (s *Service) Join(ctx context.Context, sc scope.Scope, userID int, id int64) (*dto.JoinView, error) {
	l, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if l.RoomName == nil || *l.RoomName == "" {
		return nil, model.ErrLiveKitUnavailable
	}
	if l.Status != model.StatusLive {
		return nil, model.ErrNotLive
	}
	if !s.liveKitReady() {
		return nil, model.ErrLiveKitUnavailable
	}

	canPublish := s.canPublish(ctx, userID)
	displayName := s.repo.DisplayNameForUser(ctx, sc.CustomerID, userID)

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
	return &dto.JoinView{
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

func (s *Service) Attendance(ctx context.Context, sc scope.Scope, id int64) ([]dto.AttendanceView, error) {
	if _, err := s.repo.Get(ctx, sc, id); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListAttendance(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AttendanceView, 0, len(rows))
	for i := range rows {
		out = append(out, dto.AttendanceViewOf(&rows[i]))
	}
	return out, nil
}

func (s *Service) RecordingURL(ctx context.Context, sc scope.Scope, id int64) (string, error) {
	l, err := s.repo.Get(ctx, sc, id)
	if err != nil {
		return "", err
	}
	if l.RecordingURL == nil || *l.RecordingURL == "" {
		return "", model.ErrNoRecording
	}
	return *l.RecordingURL, nil
}
