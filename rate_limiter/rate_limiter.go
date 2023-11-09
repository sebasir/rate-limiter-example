package ratelimiter

import (
	"fmt"
	"github.com/go-redis/redis"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
	"time"
)

var InternalErrorResult = &pb.Result{
	Status:          pb.Status_ERROR,
	ResponseMessage: "internal server error",
}

type client struct {
	rdb      *redis.Client
	delegate service.Client
	logger   *zap.Logger
}

func NewClient(rdb *redis.Client, delegate service.Client) service.Client {
	return &client{
		rdb:      rdb,
		delegate: delegate,
		logger:   zap.L(),
	}
}

func (c client) Send(mail *pb.Notification, config *pb.Config) (*pb.Result, error) {
	c.logger.Debug("sending notification...", zap.String("email", mail.Recipient))

	key := fmt.Sprintf("%s:%s", mail.Recipient, config.Name)
	intCmd := c.rdb.Incr(key)

	count, err := intCmd.Result()
	if err != nil {
		c.logger.Error("error trying to persist count in cache", zap.Error(err))
		return InternalErrorResult, err
	}

	var ttl time.Duration
	timeWindow := config.Unit
	if count == 1 {
		boolCmd := c.rdb.Expire(key, timeWindow.AsDuration())
		_, err = boolCmd.Result()
		if err != nil {
			c.logger.Error("error trying to submit expiration", zap.String("key", key))
			return InternalErrorResult, err
		}
		ttl = timeWindow.AsDuration()
	} else {
		durationCmd := c.rdb.TTL(key)
		ttl, err = durationCmd.Result()
		if err != nil {
			c.logger.Error("error trying to acquire current TTL", zap.String("key", key))
			return InternalErrorResult, err
		}
	}

	if count > config.Limit {
		return &pb.Result{
			Status:          pb.Status_REJECTED,
			ResponseMessage: "notification to recipient was rejected",
		}, nil
	}

	c.logger.Info("sending notification",
		zap.Int64("request_count", count),
		zap.String("recipient", mail.Recipient),
		zap.String("notification_config", config.Name),
		zap.Duration("ttl", ttl))

	return c.delegate.Send(mail, config)
}
