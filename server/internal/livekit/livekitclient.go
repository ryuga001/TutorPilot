package livekit

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/livekit/protocol/auth"
	lkprotocol "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"tutorpilot/internal/pkg/storage"
)

var ErrUnavailable = errors.New("livekit service is unavailable")

type LiveKitClient struct {
	Room   *lksdk.RoomServiceClient
	Egress *lksdk.EgressClient

	apiKey    string
	apiSecret string
}

func New(url, apiKey, apiSecret string) *LiveKitClient {
	return &LiveKitClient{
		Room: lksdk.NewRoomServiceClient(
			url,
			apiKey,
			apiSecret,
		),
		Egress: lksdk.NewEgressClient(
			url,
			apiKey,
			apiSecret,
		),
		apiKey:    apiKey,
		apiSecret: apiSecret,
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
	return c != nil && c.Room != nil && c.Egress != nil
}

func (c *LiveKitClient) CreateRoom(ctx context.Context, roomName string) error {
	if c == nil || c.Room == nil {
		return ErrUnavailable
	}
	out, ok := invokeLiveKitMethod(c.Room, "CreateRoom", ctx, &lkprotocol.CreateRoomRequest{
		Name:            roomName,
		EmptyTimeout:    1800,
		MaxParticipants: 100,
	})
	if !ok {
		return ErrUnavailable
	}
	if len(out) > 1 && out[1].IsValid() && !out[1].IsNil() {
		if err, ok := out[1].Interface().(error); ok {
			return err
		}
	}
	return nil
}

func (c *LiveKitClient) DeleteRoom(ctx context.Context, roomName string) error {
	if c == nil || c.Room == nil {
		return nil
	}
	out, ok := invokeLiveKitMethod(c.Room, "DeleteRoom", ctx, &lkprotocol.DeleteRoomRequest{Room: roomName})
	if !ok {
		return nil
	}
	if len(out) > 1 && out[1].IsValid() && !out[1].IsNil() {
		if err, ok := out[1].Interface().(error); ok {
			return err
		}
	}
	return nil
}

func (c *LiveKitClient) StartRoomCompositeEgress(ctx context.Context, roomName, objectKey string, s *storage.Storage) (*lkprotocol.EgressInfo, error) {
	if c == nil || c.Egress == nil || s == nil {
		return nil, ErrUnavailable
	}

	scheme := "http"
	if s.UseSSLValue() {
		scheme = "https"
	}
	endpoint := fmt.Sprintf("%s://%s", scheme, s.Endpoint())

	out, ok := invokeLiveKitMethod(c.Egress, "StartRoomCompositeEgress", ctx, &lkprotocol.RoomCompositeEgressRequest{
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
						Endpoint:       endpoint,
						ForcePathStyle: true,
					},
				},
			},
		},
	})
	if !ok {
		return nil, ErrUnavailable
	}
	if len(out) > 1 && out[1].IsValid() && !out[1].IsNil() {
		if err, ok := out[1].Interface().(error); ok {
			return nil, err
		}
	}
	if len(out) == 0 || !out[0].IsValid() || out[0].IsNil() {
		return nil, nil
	}
	if info, ok := out[0].Interface().(*lkprotocol.EgressInfo); ok {
		return info, nil
	}
	return nil, nil
}

func (c *LiveKitClient) StopEgress(ctx context.Context, egressID string) (*lkprotocol.EgressInfo, error) {
	if c == nil || c.Egress == nil {
		return nil, ErrUnavailable
	}
	out, ok := invokeLiveKitMethod(c.Egress, "StopEgress", ctx, &lkprotocol.StopEgressRequest{EgressId: egressID})
	if !ok {
		return nil, ErrUnavailable
	}
	if len(out) > 1 && out[1].IsValid() && !out[1].IsNil() {
		if err, ok := out[1].Interface().(error); ok {
			return nil, err
		}
	}
	if len(out) == 0 || !out[0].IsValid() || out[0].IsNil() {
		return nil, nil
	}
	if info, ok := out[0].Interface().(*lkprotocol.EgressInfo); ok {
		return info, nil
	}
	return nil, nil
}

func invokeLiveKitMethod(target any, methodName string, args ...any) ([]reflect.Value, bool) {
	value := reflect.ValueOf(target)
	if !value.IsValid() {
		return nil, false
	}
	method := value.MethodByName(methodName)
	if !method.IsValid() {
		return nil, false
	}
	callArgs := make([]reflect.Value, 0, len(args))
	for _, arg := range args {
		callArgs = append(callArgs, reflect.ValueOf(arg))
	}
	return method.Call(callArgs), true
}

func (c *LiveKitClient) GenerateToken(identity, name string, ttl time.Duration, grant VideoGrant) (string, error) {
	if c == nil {
		return "", ErrUnavailable
	}
	at := auth.NewAccessToken(c.apiKey, c.apiSecret)

	canPub := grant.CanPublish
	canSub := grant.CanSubscribe
	canData := grant.CanPublishData

	at.AddGrant(&auth.VideoGrant{
		RoomJoin:       grant.RoomJoin,
		Room:           grant.Room,
		CanPublish:     &canPub,
		CanSubscribe:   &canSub,
		CanPublishData: &canData,
	})
	at.SetIdentity(identity)
	at.SetName(name)
	at.SetValidFor(ttl)

	return at.ToJWT()
}
