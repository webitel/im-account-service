package updates

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/webitel/im-account-service/infra/pubsub/factory/amqp"
)

const (
  ContentTypeHeader = "content-type" // MIME type ; encoding info
  MessageTypeHeader = "x-proto-type" // .well-known content structure
)

// Indicates that recv.(*Message) is a Publication [FROM] your running service code around ..
func IsSelfUpdate(recv *message.Message) bool {
  // TODO: all drivers supported variants ..
  return amqp.IsSelfPublication(recv)
}