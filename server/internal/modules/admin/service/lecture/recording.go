package lecture

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	model "tutorpilot/internal/modules/admin/model/lecture"
	"tutorpilot/internal/pkg/pg"
)

const RecordingsFolder = "Lecture Recordings"

type DriveWriter interface {
	EnsureFolder(ctx context.Context, q pg.Querier, batchID int, parentID *int, name string, isSystem bool) (int, error)
	InsertFile(ctx context.Context, q pg.Querier, batchID int, parentID *int, name, objectKey, contentType string, sizeBytes int64) (int, error)
}

type ObjectURLFunc func(objectKey string) string

func (s *Service) SetDrive(drive DriveWriter, objectURL ObjectURLFunc) {
	s.drive = drive
	s.objectURL = objectURL
}

func (s *Service) CompleteRecording(
	ctx context.Context,
	roomName, objectKey string,
	sizeBytes int64,
	duration time.Duration,
) error {
	if s.drive == nil {
		return errors.New("drive is not configured")
	}

	tx, err := s.repo.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	l, err := s.repo.ByRoomName(ctx, tx, roomName, true)
	if err != nil {
		return err
	}
	if l.RecordingNodeID != nil {
		return nil
	}

	if objectKey == "" {
		if l.RecordingObjectKey == nil || *l.RecordingObjectKey == "" {
			return errors.New("recording has no object key")
		}
		objectKey = *l.RecordingObjectKey
	}

	recordingsID, err := s.drive.EnsureFolder(ctx, tx, l.BatchID, nil, RecordingsFolder, true)
	if err != nil {
		return fmt.Errorf("ensure recordings folder: %w", err)
	}
	lectureFolderID, err := s.drive.EnsureFolder(ctx, tx, l.BatchID, &recordingsID, lectureFolderName(l), false)
	if err != nil {
		return fmt.Errorf("ensure lecture folder: %w", err)
	}

	nodeID, err := s.drive.InsertFile(ctx, tx, l.BatchID, &lectureFolderID,
		recordingFileName(objectKey), objectKey, "video/mp4", sizeBytes)
	if err != nil {
		return fmt.Errorf("insert recording node: %w", err)
	}

	url := objectKey
	if s.objectURL != nil {
		url = s.objectURL(objectKey)
	}

	attached, err := s.repo.AttachRecording(ctx, tx, l.ID, nodeID,
		url, int(duration.Seconds()), sizeBytes)
	if err != nil {
		return err
	}
	if !attached {
		return nil
	}
	return tx.Commit(ctx)
}

func (s *Service) MarkRecordingFailed(ctx context.Context, roomName string) error {
	l, err := s.repo.ByRoomName(ctx, nil, roomName, false)
	if err != nil {
		return err
	}
	return s.repo.SetRecordingStatus(ctx, nil, l.ID, model.RecordingFailed)
}

func (s *Service) MarkRecordingStarted(ctx context.Context, roomName string) error {
	l, err := s.repo.ByRoomName(ctx, nil, roomName, false)
	if err != nil {
		return err
	}
	return s.repo.SetRecordingStatus(ctx, nil, l.ID, model.RecordingRecording)
}

func (s *Service) RoomFinished(ctx context.Context, roomName string) error {
	l, err := s.repo.ByRoomName(ctx, nil, roomName, false)
	if err != nil {
		return err
	}
	if err := s.repo.CloseOpenAttendance(ctx, l.ID); err != nil {
		log.Printf("lecture %d: could not close attendance: %v", l.ID, err)
	}
	return s.repo.ForceEnd(ctx, l.ID)
}

func (s *Service) ParticipantJoined(ctx context.Context, roomName string, userID int, displayName string) error {
	l, err := s.repo.ByRoomName(ctx, nil, roomName, false)
	if err != nil {
		return err
	}
	if displayName == "" {
		displayName = s.repo.DisplayNameForUser(ctx, l.CustomerID, userID)
	}
	return s.repo.OpenAttendance(ctx, l.ID, userID, displayName)
}

func (s *Service) ParticipantLeft(ctx context.Context, roomName string, userID int) error {
	l, err := s.repo.ByRoomName(ctx, nil, roomName, false)
	if err != nil {
		return err
	}
	return s.repo.CloseAttendance(ctx, l.ID, userID)
}

func lectureFolderName(l *model.Lecture) string {
	title := sanitizeName(l.Title)
	if title == "" {
		title = "lecture"
	}
	return fmt.Sprintf("%s (#%d)", title, l.ID)
}

func recordingFileName(objectKey string) string {
	name := path.Base(objectKey)
	if name == "" || name == "." || name == "/" {
		return "recording.mp4"
	}
	return name
}

func sanitizeName(s string) string {
	replaced := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\n', '\r', '\t':
			return ' '
		}
		return r
	}, s)
	return strings.TrimSpace(strings.Join(strings.Fields(replaced), " "))
}
