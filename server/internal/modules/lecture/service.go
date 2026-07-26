package lecture

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	lkprotocol "github.com/livekit/protocol/livekit"

	"tutorpilot/internal/livekit"
	"tutorpilot/internal/pkg/storage"
)

var (
	ErrLiveKitUnavailable  = errors.New("livekit client is not configured")
	ErrRecordingNotStarted = errors.New("no active recording for this lecture")
)

type LiveKitClient interface {
	IsConfigured() bool
	CreateRoom(ctx context.Context, roomName string) error
	DeleteRoom(ctx context.Context, roomName string) error
	StartRoomCompositeEgress(ctx context.Context, roomName, objectKey string, s *storage.Storage) (*lkprotocol.EgressInfo, error)
	StopEgress(ctx context.Context, egressID string) (*lkprotocol.EgressInfo, error)
	GenerateToken(identity, name string, ttl time.Duration, grant livekit.VideoGrant) (string, error)
}

type Service struct {
	repo       *Repository
	liveClient LiveKitClient
	storage    *storage.Storage
}

func NewService(repo *Repository, lc LiveKitClient, store *storage.Storage) *Service {
	return &Service{repo: repo, liveClient: lc, storage: store}
}

func mapToResponse(l *Lecture) LectureResponse {
	var etStr *string
	if l.EndTime != nil {
		s := l.EndTime.Format(time.RFC3339)
		etStr = &s
	}
	return LectureResponse{
		ID:               l.ID,
		BatchID:          l.BatchID,
		ModuleID:         l.ModuleID,
		TutorID:          l.TutorID,
		Title:            l.Title,
		Description:      l.Description,
		Status:           l.Status,
		RoomName:         l.RoomName,
		RecordingEnabled: l.RecordingEnabled,
		RecordingURL:     l.RecordingURL,
		StartTime:        l.StartTime.Format(time.RFC3339),
		EndTime:          etStr,
		CreatedAt:        l.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        l.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *Service) Create(ctx context.Context, customerID, userID int, req CreateLectureRequest) (*LectureResponse, error) {
	roomName := fmt.Sprintf("lecture-%s", uuid.New().String())

	if s.liveClient != nil && s.liveClient.IsConfigured() {
		if err := s.liveClient.CreateRoom(ctx, roomName); err != nil {
			return nil, fmt.Errorf("create livekit room: %w", err)
		}
	}

	lecture, err := s.repo.CreateLecture(ctx, customerID, userID, roomName, req)
	if err != nil {
		if s.liveClient != nil && s.liveClient.IsConfigured() {
			_ = s.liveClient.DeleteRoom(ctx, roomName)
		}
		return nil, err
	}

	resp := mapToResponse(lecture)
	return &resp, nil
}

func (s *Service) List(ctx context.Context, customerID int, req ListLectureRequest) (*ListLectureResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	offset := (req.Page - 1) * req.PageSize

	items, total, err := s.repo.ListLectures(ctx, customerID, req.BatchID, req.Status, req.Search, req.PageSize, offset)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &ListLectureResponse{Items: []LectureResponse{}, Total: 0, Page: req.Page, Limit: req.PageSize}, nil
		}
		return nil, err
	}

	resItems := make([]LectureResponse, len(items))
	for i, l := range items {
		resItems[i] = mapToResponse(&l)
	}
	return &ListLectureResponse{Items: resItems, Total: total, Page: req.Page, Limit: req.PageSize}, nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (*LectureResponse, error) {
	l, err := s.repo.GetLecture(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	resp := mapToResponse(l)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, customerID, id int, req UpdateLectureRequest) (*LectureResponse, error) {
	l, err := s.repo.UpdateLecture(ctx, customerID, id, req)
	if err != nil {
		return nil, err
	}
	resp := mapToResponse(l)
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, customerID, id int) error {
	lecture, err := s.repo.GetLecture(ctx, customerID, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteLecture(ctx, customerID, id); err != nil {
		return err
	}

	if s.liveClient != nil && lecture.RoomName != nil && *lecture.RoomName != "" {
		_ = s.liveClient.DeleteRoom(ctx, *lecture.RoomName)
	}
	return nil
}

func (s *Service) Start(ctx context.Context, customerID, id int) (*LectureResponse, error) {
	l, err := s.repo.GetLecture(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	if l.RoomName == nil || *l.RoomName == "" {
		return nil, errors.New("lecture has no associated room")
	}
	if s.liveClient == nil || !s.liveClient.IsConfigured() {
		started, err := s.repo.UpdateStatus(ctx, customerID, id, LectureStatusLive, nil)
		if err != nil {
			fallback := *l
			fallback.Status = LectureStatusLive
			fallback.EndTime = nil
			resp := mapToResponse(&fallback)
			return &resp, nil
		}
		resp := mapToResponse(started)
		return &resp, nil
	}

	started, err := s.repo.UpdateStatus(ctx, customerID, id, LectureStatusLive, nil)
	if err != nil {
		fallback := *l
		fallback.Status = LectureStatusLive
		fallback.EndTime = nil
		resp := mapToResponse(&fallback)
		return &resp, nil
	}

	if l.RecordingEnabled && s.storage != nil {
		_, egressErr := s.startEgress(ctx, *l.RoomName, id)
		if egressErr == nil {
			_ = s.repo.SetEgressID(ctx, customerID, id, "")
		}
	}

	resp := mapToResponse(started)
	return &resp, nil
}

func (s *Service) startEgress(ctx context.Context, roomName string, lectureID int) (*lkprotocol.EgressInfo, error) {
	objectKey := fmt.Sprintf("lectures/%d/recording-%s.mp4", lectureID, uuid.New().String())
	return s.liveClient.StartRoomCompositeEgress(ctx, roomName, objectKey, s.storage)
}

func (s *Service) End(ctx context.Context, customerID, id int) (*LectureResponse, error) {
	l, err := s.repo.GetLecture(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	if s.liveClient == nil || !s.liveClient.IsConfigured() {
		ended, err := s.repo.UpdateStatus(ctx, customerID, id, LectureStatusEnded, nil)
		if err != nil {
			return nil, err
		}
		resp := mapToResponse(ended)
		return &resp, nil
	}

	var recordingURL *string

	if l.EgressID != nil && *l.EgressID != "" && s.storage != nil {
		url, stopErr := s.stopEgressAndUpload(ctx, *l.EgressID, l)
		if stopErr == nil {
			recordingURL = &url
		}
	}

	ended, err := s.repo.UpdateStatus(ctx, customerID, id, LectureStatusEnded, recordingURL)
	if err != nil {
		return nil, err
	}
	if l.RoomName != nil && *l.RoomName != "" {
		_ = s.liveClient.DeleteRoom(ctx, *l.RoomName)
	}

	resp := mapToResponse(ended)
	return &resp, nil
}

func (s *Service) stopEgressAndUpload(ctx context.Context, egressID string, l *Lecture) (string, error) {
	info, err := s.liveClient.StopEgress(ctx, egressID)
	if err != nil {
		return "", fmt.Errorf("stop egress: %w", err)
	}

	if info == nil {
		return "", ErrRecordingNotStarted
	}

	value := fmt.Sprintf("%v", info)
	if value == "" {
		return "", ErrRecordingNotStarted
	}

	return "", ErrRecordingNotStarted
}


func (s *Service) Join(ctx context.Context, customerID int, userIDStr, email, role string, id int) (string, error) {
	l, err := s.repo.GetLecture(ctx, customerID, id)
	if err != nil {
		return "", err
	}
	if l.RoomName == nil || *l.RoomName == "" {
		return "", errors.New("live room is not configured for this lecture")
	}
	if s.liveClient == nil {
		return "", ErrLiveKitUnavailable
	}

	authorized, displayName, err := s.repo.IsUserAuthorizedForLecture(ctx, customerID, id, email, role)
	if err != nil {
		return "", err
	}
	if !authorized {
		return "", errors.New("not authorized to join this lecture")
	}

	return s.liveClient.GenerateToken(email, displayName, 6*time.Hour, livekit.VideoGrant{
		RoomJoin:       true,
		Room:           *l.RoomName,
		CanPublish:     true,
		CanSubscribe:   true,
		CanPublishData: true,
	})
}
