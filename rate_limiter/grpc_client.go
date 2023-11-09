package ratelimiter

import (
	"context"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
)

type grpcClient struct {
	serviceClient pb.NotificationServiceClient
	logger        *zap.Logger
}

func NewGRPCClient(serviceClient pb.NotificationServiceClient) service.Client {
	return &grpcClient{
		serviceClient: serviceClient,
		logger:        zap.L(),
	}
}

func (c grpcClient) Send(notification *pb.Notification, config *pb.Config) (*pb.Result, error) {
	request := &pb.NotificationRequest{
		Config:       config,
		Notification: notification,
	}

	response, err := c.serviceClient.Send(context.Background(), request)
	if err != nil {
		c.logger.Error("error sending message over gRPC client", zap.Error(err))
		return InternalErrorResult, err
	}

	return response.Result, nil
}
