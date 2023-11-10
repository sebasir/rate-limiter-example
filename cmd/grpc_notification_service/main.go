package main

import (
	"github.com/sebasir/rate-limiter-example/config"
	"github.com/sebasir/rate-limiter-example/http"
	"github.com/sebasir/rate-limiter-example/mail"
	"github.com/sebasir/rate-limiter-example/notification"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"strconv"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	if val := os.Getenv("DEBUG"); val == "1" {
		zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	}
}

func main() {
	logger := zap.L()
	cfg := &config.AppConfig{}
	if err := cfg.Load(); err != nil {
		log.Fatalf("error retrieving env variables: %v", err)
	}

	logger.Info("starting gRCP/HTTP Notification app")

	gRPCServerAddress := config.FormatAddress("0.0.0.0", cfg.NotificationGRPCPort)
	logger.Debug("gRPC server", zap.String("address", gRPCServerAddress))

	client := mail.NewClient()
	go func() {
		if err := os.Setenv("PORT", strconv.Itoa(cfg.NotificationHTTPPort)); err != nil {
			logger.Fatal("error setting GIN port env variable", zap.Error(err))
		}

		controller := http.NewController(client)
		logger.Debug("starting GIN HTTP server", zap.Int("port", cfg.NotificationHTTPPort))
		if err := controller.StartServer(); err != nil {
			logger.Fatal("error starting HTTP server", zap.Error(err), zap.Int("port", cfg.NotificationHTTPPort))
		}
	}()

	lis, err := net.Listen("tcp", gRPCServerAddress)
	if err != nil {
		logger.Fatal("error opening TCP channel", zap.Error(err), zap.String("address", gRPCServerAddress))
	}

	s := grpc.NewServer()
	server := notification.NewServer(client)
	pb.RegisterNotificationServiceServer(s, server)
	logger.Debug("starting gRCP server", zap.String("address", gRPCServerAddress))
	if err = s.Serve(lis); err != nil {
		logger.Fatal("error when serving on gRPC channel", zap.Error(err), zap.Int("port", cfg.NotificationGRPCPort))
	}
}
