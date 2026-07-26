package livekit

import (
	"time"

	"github.com/livekit/protocol/auth"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

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

func (c *LiveKitClient) GenerateToken(identity, name string, ttl time.Duration, grant VideoGrant) (string, error) {
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
