package updates

import (
	// "context"

	// "github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
)

// type Publication struct {
// 	Topic string
// 	message.Message
// 	context.Context
// }

// Update Arguments basic
type Update interface {
	isUpdate()

	proto.Message
	// Topic returns Update.(self) routing key for publishing ..
	Topic(string) string
	
  // Publication(ctx context.Context) (Publication, error)
}