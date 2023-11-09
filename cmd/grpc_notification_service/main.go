package main

import (
	"github.com/sebasir/rate-limiter-example/http"
	"github.com/sebasir/rate-limiter-example/mail"
	"github.com/sebasir/rate-limiter-example/notification"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

func main() {
	const address = "0.0.0.0:8281"
	logger := zap.L()

	client := mail.NewClient()
	go func() {
		controller := http.NewController(client)
		if err := controller.StartServer(); err != nil {
			logger.Fatal("error starting HTTP server", zap.Error(err), zap.String("address", address))
		}
	}()

	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Fatal("error opening TCP channel", zap.Error(err), zap.String("address", address))
	}

	logger.Info("TCP channel listening", zap.String("address", address))

	s := grpc.NewServer()
	server := notification.NewServer(client)
	pb.RegisterNotificationServiceServer(s, server)
	if err = s.Serve(lis); err != nil {
		logger.Fatal("error when serving", zap.Error(err), zap.String("address", address))
	}
}
