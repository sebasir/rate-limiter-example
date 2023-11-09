package notification

import (
	"context"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
)

type Server struct {
	pb.NotificationServiceServer
	notificationClient service.Client
	logger             *zap.Logger
}

func NewServer(client service.Client) *Server {
	return &Server{
		notificationClient: client,
		logger:             zap.L(),
	}
}

func (s *Server) Send(_ context.Context, request *pb.NotificationRequest) (*pb.NotificationResponse, error) {
	result, err := s.notificationClient.Send(request.GetNotification(), request.GetConfig())
	if err != nil {
		return nil, err
	}

	return &pb.NotificationResponse{
		Result: result,
	}, nil
}
