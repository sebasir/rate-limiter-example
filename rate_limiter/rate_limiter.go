package ratelimiter

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/sebasir/rate-limiter-example/manager"
	"github.com/sebasir/rate-limiter-example/model"
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
	manager  manager.Service
	logger   *zap.Logger
}

func NewClient(rdb *redis.Client, delegate service.Client, manager manager.Service) service.ExtendedClient {
	return &client{
		rdb:      rdb,
		delegate: delegate,
		manager:  manager,
		logger:   zap.L(),
	}
}

func (c client) Send(n *pb.Notification) (*pb.Result, error) {
	c.logger.Debug("sending notification", zap.String("recipient", n.Recipient))

	config, err := c.manager.GetByName(n.NotificationType)
	if err != nil {
		c.logger.Error("error trying to fetch notification type configuration",
			zap.Error(err),
			zap.String("notification_type", n.NotificationType))
		return InternalErrorResult, err
	}

	key := fmt.Sprintf("%s:%s", n.Recipient, n.NotificationType)
	intCmd := c.rdb.Incr(key)

	count, err := intCmd.Result()
	if err != nil {
		c.logger.Error("error trying to persist count in cache", zap.Error(err))
		return InternalErrorResult, err
	}

	var ttl time.Duration
	timeWindow := config.Unit
	if count == 1 {
		boolCmd := c.rdb.Expire(key, timeWindow)
		_, err = boolCmd.Result()
		if err != nil {
			c.logger.Error("error trying to submit expiration", zap.String("key", key))
			return InternalErrorResult, err
		}
		ttl = timeWindow
	} else {
		durationCmd := c.rdb.TTL(key)
		ttl, err = durationCmd.Result()
		if err != nil {
			c.logger.Error("error trying to acquire current TTL", zap.String("key", key))
			return InternalErrorResult, err
		}
	}

	if count > config.Limit {
		c.logger.Debug("rejecting notification",
			zap.Int64("request_count", count),
			zap.String("recipient", n.Recipient),
			zap.String("notification_config", config.Name),
			zap.Duration("ttl", ttl))
		return &pb.Result{
			Status:          pb.Status_REJECTED,
			ResponseMessage: "notification to recipient was rejected",
		}, nil
	}

	c.logger.Debug("sending notification to gRPC delegate",
		zap.Int64("request_count", count),
		zap.String("recipient", n.Recipient),
		zap.String("notification_config", config.Name),
		zap.Duration("ttl", ttl))

	return c.delegate.Send(n)
}

func (c client) ListNotificationConfig() ([]*model.Config, error) {
	return c.manager.ListNotificationConfig()
}
