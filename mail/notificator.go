package mail

import (
	"fmt"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
)

type client struct {
	logger *zap.Logger
}

func NewClient() service.Client {
	return &client{
		logger: zap.L(),
	}
}

func (c client) Send(notification *pb.Notification) (*pb.Result, error) {
	c.logger.Debug("sending notification to recipient", zap.String("recipient", notification.Recipient))

	// send email...

	c.logger.Debug("notification sent to recipient", zap.String("recipient", notification.Recipient))
	return &pb.Result{
		Status:          pb.Status_SENT,
		ResponseMessage: fmt.Sprintf("notification sent to recipient (%s)", notification.Recipient),
	}, nil
}
