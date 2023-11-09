package service

import (
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
)

type Client interface {
	Send(mail *pb.Notification, config *pb.Config) (*pb.Result, error)
}
