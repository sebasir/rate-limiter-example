package main

import (
	"github.com/go-redis/redis"
	"github.com/sebasir/rate-limiter-example/http"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	ratelimiter "github.com/sebasir/rate-limiter-example/rate_limiter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

func main() {
	const address = "localhost:8281"
	logger := zap.L()
	rdb := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})

	if err := rdb.FlushDB().Err(); err != nil {
		logger.Fatal("error connecting to Redis server", zap.Error(err), zap.String("address", "6379"))
	}

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("error dialing to gRPC server", zap.Error(err), zap.String("address", address))
	}

	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			logger.Fatal("error closing gRPC client", zap.Error(err))
		}
	}(conn)

	c := pb.NewNotificationServiceClient(conn)

	delegate := ratelimiter.NewGRPCClient(c)
	client := ratelimiter.NewClient(rdb, delegate)
	controller := http.NewController(client)
	if err = controller.StartServer(); err != nil {
		logger.Fatal("error when serving HTTP", zap.Error(err), zap.String("address", address))
	}
}
