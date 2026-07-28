package lecture

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	"tutorpilot/internal/pkg/pg"
)

// RecordingsFolder is the reserved folder every batch's recordings are filed into.
const RecordingsFolder = "Lecture Recordings"

// DriveWriter is the slice of the batch drive the recording pipeline needs. The
// batches module implements it; taking a pg.Querier lets these run inside the same
// transaction as the lectures update, so a recording is never half-filed.
type DriveWriter interface {
	EnsureFolder(ctx context.Context, q pg.Querier, batchID int, parentID *int, name string, isSystem bool) (int, error)
	InsertFile(ctx context.Context, q pg.Querier, batchID int, parentID *int, name, objectKey, contentType string, sizeBytes int64) (int, error)
}

// ObjectURLFunc resolves a stored object key to a URL a browser can play.
type ObjectURLFunc func(objectKey string) string

// SetDrive supplies the drive dependencies. They arrive after construction because
// the batches module and this one are built independently in server.go.
func (s *Service) SetDrive(drive DriveWriter, objectURL ObjectURLFunc) {
	s.drive = drive
	s.objectURL = objectURL
}

// CompleteRecording files a finished recording into the batch drive and links it to
// the lecture, in one transaction.
//
// It is idempotent, which is not optional: LiveKit retries webhook deliveries, so a
// second call must not produce a second copy. The lecture row is locked and the
// AttachRecording update is guarded on recording_node_id IS NULL, so a duplicate
// delivery rolls back having changed nothing.
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
		// Already filed by an earlier delivery of this event.
		return nil
	}

	// Prefer the key the egress actually wrote; fall back to the one reserved at
	// start, in case the event omits it.
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
		// Another delivery won; discard this one's inserts.
		return nil
	}
	return tx.Commit(ctx)
}

// MarkRecordingFailed records that the pipeline gave up, so the UI can say so
// instead of showing a player that will never load.
func (s *Service) MarkRecordingFailed(ctx context.Context, roomName string) error {
	l, err := s.repo.ByRoomName(ctx, nil, roomName, false)
	if err != nil {
		return err
	}
	return s.repo.SetRecordingStatus(ctx, nil, l.ID, RecordingFailed)
}

// MarkRecordingStarted confirms egress actually began.
func (s *Service) MarkRecordingStarted(ctx context.Context, roomName string) error {
	l, err := s.repo.ByRoomName(ctx, nil, roomName, false)
	if err != nil {
		return err
	}
	return s.repo.SetRecordingStatus(ctx, nil, l.ID, RecordingRecording)
}

// RoomFinished closes a lecture whose room ended without anyone pressing End, which
// is what rescues a lecture left permanently "live".
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

// ParticipantJoined and ParticipantLeft record attendance from LiveKit events.
// Whether the participant is a tutor, student, or admin is derived by
// OpenAttendance from their id, not trusted from the event.
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

// lectureFolderName is the per-lecture subfolder. The id is appended so two
// lectures with the same title do not collide, and so the folder is identifiable
// after a rename.
func lectureFolderName(l *Lecture) string {
	title := sanitizeName(l.Title)
	if title == "" {
		title = "lecture"
	}
	return fmt.Sprintf("%s (#%d)", title, l.ID)
}

// recordingFileName keeps the stored object's basename, which already carries a
// unique suffix from the key generated at start.
func recordingFileName(objectKey string) string {
	name := path.Base(objectKey)
	if name == "" || name == "." || name == "/" {
		return "recording.mp4"
	}
	return name
}

// sanitizeName strips the characters that make a drive entry awkward to display or
// address, without transliterating the title away.
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
