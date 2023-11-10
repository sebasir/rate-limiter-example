package main

import (
	"github.com/go-redis/redis"
	"github.com/sebasir/rate-limiter-example/config"
	"github.com/sebasir/rate-limiter-example/http"
	"github.com/sebasir/rate-limiter-example/manager"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	ratelimiter "github.com/sebasir/rate-limiter-example/rate_limiter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		logger.Fatal("error retrieving env variables", zap.Error(err))
	}

	logger.Info("starting Rate Limiter app")

	redisAddress := config.FormatAddress(cfg.RedisHost, cfg.RedisPort)
	logger.Debug("redis server", zap.String("address", redisAddress))
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddress,
	})

	logger.Debug("connecting and flushing redis cache", zap.String("address", redisAddress))
	if err := rdb.FlushDB().Err(); err != nil {
		logger.Fatal("error connecting to Redis server", zap.Error(err), zap.String("address", redisAddress))
	}

	grpcServerAddress := config.FormatAddress(cfg.NotificationHost, cfg.NotificationGRPCPort)
	logger.Debug("dialing to gRPC notification server", zap.String("address", grpcServerAddress))
	conn, err := grpc.Dial(grpcServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("error dialing to gRPC notification server", zap.Error(err), zap.String("address", grpcServerAddress))
	}

	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			logger.Fatal("error closing gRPC client", zap.Error(err))
		}
	}(conn)

	c := pb.NewNotificationServiceClient(conn)

	if err := os.Setenv("PORT", strconv.Itoa(cfg.RateLimiterHttpPort)); err != nil {
		logger.Fatal("error setting GIN port env variable", zap.Error(err))
	}

	mgr := manager.NewClient(rdb)
	delegate := ratelimiter.NewGRPCClient(c)
	client := ratelimiter.NewClient(rdb, delegate, mgr)
	controller := http.NewControllerWithConfig(client)
	logger.Debug("starting GIN HTTP server", zap.Int("port", cfg.RateLimiterHttpPort))
	if err = controller.StartServer(); err != nil {
		logger.Fatal("error when serving HTTP", zap.Error(err), zap.Int("port", cfg.RateLimiterHttpPort))
	}
}
