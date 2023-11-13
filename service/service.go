package service

import (
	"github.com/sebasir/rate-limiter-example/model"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
)

type Client interface {
	Send(notification *pb.Notification) (*pb.Result, error)
}

type ConfigClient interface {
	ListNotificationConfig() ([]*model.Config, error)
	PersistNotificationConfig(*model.Config) error
}

type ExtendedClient interface {
	Client
	ConfigClient
}
