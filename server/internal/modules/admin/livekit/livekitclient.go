package livekit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/livekit/protocol/auth"
	lkprotocol "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"tutorpilot/internal/modules/admin/storage"
)

var ErrUnavailable = errors.New("livekit service is unavailable")

type LiveKitClient struct {
	Room   *lksdk.RoomServiceClient
	Egress *lksdk.EgressClient

	apiKey    string
	apiSecret string

	emptyTimeout    uint32
	maxParticipants uint32
}

type Options struct {
	URL       string
	APIKey    string
	APISecret string

	EmptyTimeout time.Duration

	MaxParticipants int
}

func New(o Options) *LiveKitClient {
	emptyTimeout := uint32(o.EmptyTimeout.Seconds())
	if emptyTimeout == 0 {
		emptyTimeout = 300
	}
	maxParticipants := uint32(o.MaxParticipants)
	if maxParticipants == 0 {
		maxParticipants = 100
	}
	return &LiveKitClient{
		Room:            lksdk.NewRoomServiceClient(o.URL, o.APIKey, o.APISecret),
		Egress:          lksdk.NewEgressClient(o.URL, o.APIKey, o.APISecret),
		apiKey:          o.APIKey,
		apiSecret:       o.APISecret,
		emptyTimeout:    emptyTimeout,
		maxParticipants: maxParticipants,
	}
}

type VideoGrant struct {
	RoomJoin       bool
	Room           string
	CanPublish     bool
	CanSubscribe   bool
	CanPublishData bool
}

func (c *LiveKitClient) IsConfigured() bool {
	return c != nil && c.Room != nil && c.Egress != nil && c.apiSecret != ""
}

func (c *LiveKitClient) CreateRoom(ctx context.Context, roomName, metadata string) error {
	if !c.IsConfigured() {
		return ErrUnavailable
	}
	_, err := c.Room.CreateRoom(ctx, &lkprotocol.CreateRoomRequest{
		Name:            roomName,
		EmptyTimeout:    c.emptyTimeout,
		MaxParticipants: c.maxParticipants,
		Metadata:        metadata,
	})
	return err
}

func (c *LiveKitClient) DeleteRoom(ctx context.Context, roomName string) error {
	if !c.IsConfigured() {
		return ErrUnavailable
	}
	_, err := c.Room.DeleteRoom(ctx, &lkprotocol.DeleteRoomRequest{Room: roomName})
	return err
}

func (c *LiveKitClient) StartRoomCompositeEgress(
	ctx context.Context,
	roomName, objectKey string,
	s *storage.Storage,
) (*lkprotocol.EgressInfo, error) {
	if !c.IsConfigured() || s == nil {
		return nil, ErrUnavailable
	}

	scheme := "http"
	if s.UseSSLValue() {
		scheme = "https"
	}

	return c.Egress.StartRoomCompositeEgress(ctx, &lkprotocol.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   "speaker",
		Output: &lkprotocol.RoomCompositeEgressRequest_File{
			File: &lkprotocol.EncodedFileOutput{
				FileType: lkprotocol.EncodedFileType_MP4,
				Filepath: objectKey,
				Output: &lkprotocol.EncodedFileOutput_S3{
					S3: &lkprotocol.S3Upload{
						AccessKey:      s.AccessKeyValue(),
						Secret:         s.SecretKeyValue(),
						Region:         "us-east-1",
						Bucket:         s.BucketName(),
						Endpoint:       fmt.Sprintf("%s://%s", scheme, s.Endpoint()),
						ForcePathStyle: true,
					},
				},
			},
		},
	})
}

func (c *LiveKitClient) StopEgress(ctx context.Context, egressID string) (*lkprotocol.EgressInfo, error) {
	if !c.IsConfigured() {
		return nil, ErrUnavailable
	}
	return c.Egress.StopEgress(ctx, &lkprotocol.StopEgressRequest{EgressId: egressID})
}

func (c *LiveKitClient) GenerateToken(identity, name, metadata string, ttl time.Duration, grant VideoGrant) (string, error) {
	if c == nil || c.apiSecret == "" {
		return "", ErrUnavailable
	}
	at := auth.NewAccessToken(c.apiKey, c.apiSecret)

	canPub := grant.CanPublish
	canSub := grant.CanSubscribe
	canData := grant.CanPublishData
	canUpdateOwn := true

	at.AddGrant(&auth.VideoGrant{
		RoomJoin:             grant.RoomJoin,
		Room:                 grant.Room,
		CanPublish:           &canPub,
		CanSubscribe:         &canSub,
		CanPublishData:       &canData,
		CanUpdateOwnMetadata: &canUpdateOwn,
	})
	at.SetIdentity(identity)
	at.SetName(name)
	at.SetMetadata(metadata)
	at.SetValidFor(ttl)

	return at.ToJWT()
}
