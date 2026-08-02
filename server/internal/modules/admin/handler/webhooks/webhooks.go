package webhooks

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	lkauth "github.com/livekit/protocol/auth"
	lkprotocol "github.com/livekit/protocol/livekit"
	lkwebhook "github.com/livekit/protocol/webhook"

	lecture "tutorpilot/internal/modules/admin/service/lecture"
)

type Deps struct {
	Lectures  *lecture.Service
	APIKey    string
	APISecret string
}

type Handler struct {
	lectures *lecture.Service
	keys     lkauth.KeyProvider
}

func NewHandler(d Deps) *Handler {
	var keys lkauth.KeyProvider
	if d.APIKey != "" && d.APISecret != "" {
		keys = lkauth.NewSimpleKeyProvider(d.APIKey, d.APISecret)
	}
	return &Handler{lectures: d.Lectures, keys: keys}
}

type participantMetadata struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
}

func (h *Handler) LiveKit(c *gin.Context) {
	if h.keys == nil {
		log.Print("webhooks: livekit callback received but no API secret is configured")
		c.Status(http.StatusServiceUnavailable)
		return
	}

	event, err := lkwebhook.ReceiveWebhookEvent(c.Request, h.keys)
	if err != nil {
		if errors.Is(err, lkwebhook.ErrNoAuthHeader) || errors.Is(err, lkwebhook.ErrInvalidChecksum) {
			c.Status(http.StatusUnauthorized)
			return
		}
		log.Printf("webhooks: could not read livekit event: %v", err)
		c.Status(http.StatusBadRequest)
		return
	}

	if err := h.dispatch(c, event); err != nil {
		log.Printf("webhooks: %s for room %q: %v", event.Event, roomName(event), err)
	}
	c.Status(http.StatusOK)
}

func (h *Handler) dispatch(c *gin.Context, event *lkprotocol.WebhookEvent) error {
	ctx := c.Request.Context()
	room := roomName(event)
	if room == "" {
		return nil
	}

	switch event.Event {
	case lkwebhook.EventParticipantJoined:
		return h.participantJoined(c, room, event.Participant)

	case lkwebhook.EventParticipantLeft:
		userID, ok := participantUserID(event.Participant)
		if !ok {
			return nil
		}
		return h.lectures.ParticipantLeft(ctx, room, userID)

	case lkwebhook.EventEgressStarted:
		return h.lectures.MarkRecordingStarted(ctx, room)

	case lkwebhook.EventEgressEnded, lkwebhook.EventEgressUpdated:
		return h.egressUpdate(c, room, event.EgressInfo)

	case lkwebhook.EventRoomFinished:
		return h.lectures.RoomFinished(ctx, room)

	default:

		return nil
	}
}

func (h *Handler) participantJoined(c *gin.Context, room string, p *lkprotocol.ParticipantInfo) error {
	userID, ok := participantUserID(p)
	if !ok {
		return nil
	}
	meta := decodeMetadata(p)
	name := meta.Name
	if name == "" && p != nil {
		name = p.Name
	}
	return h.lectures.ParticipantJoined(c.Request.Context(), room, userID, name)
}

func (h *Handler) egressUpdate(c *gin.Context, room string, info *lkprotocol.EgressInfo) error {
	if info == nil {
		return nil
	}
	ctx := c.Request.Context()

	switch info.Status {
	case lkprotocol.EgressStatus_EGRESS_COMPLETE:
		file := firstFile(info)
		if file == nil {
			return h.lectures.MarkRecordingFailed(ctx, room)
		}

		return h.lectures.CompleteRecording(ctx, room,
			objectKeyOf(file), file.Size, time.Duration(file.Duration))

	case lkprotocol.EgressStatus_EGRESS_FAILED,
		lkprotocol.EgressStatus_EGRESS_ABORTED,
		lkprotocol.EgressStatus_EGRESS_LIMIT_REACHED:
		return h.lectures.MarkRecordingFailed(ctx, room)

	default:

		return nil
	}
}

func roomName(event *lkprotocol.WebhookEvent) string {
	if event == nil {
		return ""
	}
	if event.Room != nil && event.Room.Name != "" {
		return event.Room.Name
	}
	if event.EgressInfo != nil {
		return event.EgressInfo.RoomName
	}
	return ""
}

func firstFile(info *lkprotocol.EgressInfo) *lkprotocol.FileInfo {
	for _, f := range info.FileResults {
		if f != nil {
			return f
		}
	}
	return nil
}

func objectKeyOf(f *lkprotocol.FileInfo) string {
	if f.Filename != "" {
		return strings.TrimPrefix(f.Filename, "/")
	}
	if f.Location == "" {
		return ""
	}
	if i := strings.Index(f.Location, "://"); i >= 0 {
		rest := f.Location[i+3:]
		if j := strings.Index(rest, "/"); j >= 0 {
			return strings.TrimPrefix(rest[j+1:], "/")
		}
	}
	return ""
}

func participantUserID(p *lkprotocol.ParticipantInfo) (int, bool) {
	if p == nil {
		return 0, false
	}
	if meta := decodeMetadata(p); meta.UserID > 0 {
		return meta.UserID, true
	}
	raw, ok := strings.CutPrefix(p.Identity, "u:")
	if !ok {
		return 0, false
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func decodeMetadata(p *lkprotocol.ParticipantInfo) participantMetadata {
	var meta participantMetadata
	if p == nil || p.Metadata == "" {
		return meta
	}
	if err := json.Unmarshal([]byte(p.Metadata), &meta); err != nil {
		return participantMetadata{}
	}
	return meta
}
