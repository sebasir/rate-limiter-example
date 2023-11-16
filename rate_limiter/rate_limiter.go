package ratelimiter

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	. "github.com/sebasir/rate-limiter-example/app_errors"
	"github.com/sebasir/rate-limiter-example/manager"
	"github.com/sebasir/rate-limiter-example/model"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
	"time"
)

var InternalErrorResult = &pb.Result{
	Status:          pb.Status_INTERNAL_ERROR,
	ResponseMessage: "error processing notification request",
}

var ErrProcessingNotificationRequest = errors.New("error processing notification request")

type client struct {
	delegate service.Client
	manager  manager.Service
	rdb      redis.Cmdable
	logger   *zap.Logger
}

func NewClient(rdb redis.Cmdable, delegate service.Client, manager manager.Service) service.ExtendedClient {
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
		return InternalErrorResult, LogAndError("error trying to fetch notification type configuration",
			errors.Join(err, ErrProcessingNotificationRequest), c.logger, zap.String("notification_type", n.NotificationType))
	}

	key := fmt.Sprintf("%s:%s", n.Recipient, n.NotificationType)
	intCmd := c.rdb.Incr(key)

	count, err := intCmd.Result()
	if err != nil {
		return InternalErrorResult, LogAndError("error trying to persist count in cache",
			errors.Join(err, ErrProcessingNotificationRequest), c.logger, recipientField)
	}

	keyField := zap.String("key", key)
	var ttl time.Duration
	timeWindow := config.CalculateTime()
	if count == 1 {
		boolCmd := c.rdb.Expire(key, timeWindow)
		_, err = boolCmd.Result()
		if err != nil {
			return InternalErrorResult, LogAndError("error trying to submit expiration",
				errors.Join(err, ErrProcessingNotificationRequest), c.logger, keyField)
		}
		ttl = timeWindow
	} else {
		durationCmd := c.rdb.TTL(key)
		ttl, err = durationCmd.Result()
		if err != nil {
			return InternalErrorResult, LogAndError("error trying to acquire current TTL",
				errors.Join(err, ErrProcessingNotificationRequest), c.logger, keyField)
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
	res, err := c.delegate.Send(n)
	if err != nil {
		return InternalErrorResult, LogAndError("error trying to send notification",
			errors.Join(err, ErrProcessingNotificationRequest), c.logger, recipientField)
	}
	return res, nil
}

func (c *client) ListNotificationConfig() ([]*model.Config, error) {
	return c.manager.ListNotificationConfig()
}

func (c *client) PersistNotificationConfig(config *model.Config) error {
	return c.manager.PersistNotificationConfig(config)
}
