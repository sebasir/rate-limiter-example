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
	Status:          pb.Status_INTERNAL_ERROR,
	ResponseMessage: "internal server error",
}

type client struct {
	delegate service.Client
	manager  manager.Service
	rdb      *redis.Client
	logger   *zap.Logger
}

func NewClient(rdb *redis.Client, delegate service.Client, manager manager.Service) service.ExtendedClient {
	return &client{
		delegate: delegate,
		manager:  manager,
		rdb:      rdb,
		logger:   zap.L(),
	}
}

func (c *client) Send(n *pb.Notification) (*pb.Result, error) {
	recipientField := zap.String("recipient", n.Recipient)

	c.logger.Debug("sending notification", recipientField)

	config, err := c.manager.GetByName(n.NotificationType)
	if err != nil {
		msg := "error trying to fetch notification type configuration"
		c.logger.Error(msg, zap.Error(err), zap.String("notification_type", n.NotificationType))
		return InternalErrorResult, fmt.Errorf("%s: %w", msg, err)
	}

	key := fmt.Sprintf("%s:%s", n.Recipient, n.NotificationType)
	intCmd := c.rdb.Incr(key)

	count, err := intCmd.Result()
	if err != nil {
		msg := "error trying to persist count in cache"
		c.logger.Error(msg, zap.Error(err))
		return InternalErrorResult, fmt.Errorf("%s: %w", msg, err)
	}

	keyField := zap.String("key", key)
	var ttl time.Duration
	timeWindow := config.TimeUnit * time.Duration(config.TimeAmount)
	if count == 1 {
		boolCmd := c.rdb.Expire(key, timeWindow)
		_, err = boolCmd.Result()
		if err != nil {
			msg := "error trying to submit expiration"
			c.logger.Error(msg, keyField)
			return InternalErrorResult, fmt.Errorf("%s: %w", msg, err)
		}
		ttl = timeWindow
	} else {
		durationCmd := c.rdb.TTL(key)
		ttl, err = durationCmd.Result()
		if err != nil {
			msg := "error trying to acquire current TTL"
			c.logger.Error(msg, keyField)
			return InternalErrorResult, fmt.Errorf("%s: %w", msg, err)
		}
	}

	countField := zap.Int64("request_count", count)
	configField := zap.String("notification_config", config.Name)
	ttlField := zap.Duration("ttl", ttl)

	if count > config.LimitCount {
		c.logger.Debug("rejecting notification", countField, recipientField, configField, ttlField)
		return &pb.Result{
			Status:          pb.Status_REJECTED,
			ResponseMessage: "notification to recipient was rejected",
		}, nil
	}

	c.logger.Debug("sending notification to gRPC delegate", countField, recipientField, configField, ttlField)
	return c.delegate.Send(n)
}

func (c *client) ListNotificationConfig() ([]*model.Config, error) {
	return c.manager.ListNotificationConfig()
}

func (c *client) PersistNotificationConfig(config *model.Config) error {
	return c.manager.PersistNotificationConfig(config)
}
