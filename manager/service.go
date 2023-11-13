package manager

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/sebasir/rate-limiter-example/app_errors"
	"github.com/sebasir/rate-limiter-example/model"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
)

type Service interface {
	service.ConfigClient
	GetByName(name string) (*model.Config, error)
}

type client struct {
	rdb    *redis.Client
	logger *zap.Logger
}

func NewClient(rdb *redis.Client) Service {
	return &client{
		rdb:    rdb,
		logger: zap.L(),
	}
}

func (c *client) ListNotificationConfig() ([]*model.Config, error) {
	c.logger.Debug("retrieving notification config list")

	scanKey := key("*")
	var cursor uint64
	allKeys := make([]*model.Config, 0)
	for {
		foundKeys, cursor, err := c.rdb.Scan(cursor, scanKey, 50).Result()
		if err != nil {
			c.logger.Error("error retrieving notification config list", zap.Error(err))
			return nil, err
		}

		tempConfigs := make([]*model.Config, len(foundKeys))
		for i, foundKey := range foundKeys {
			config, err := c.getByKey(foundKey)
			if err != nil {
				return nil, err
			}
			tempConfigs[i] = config
		}

		allKeys = append(allKeys, tempConfigs...)

		if cursor == 0 {
			break
		}
	}

	return allKeys, nil
}

func (c *client) GetByName(name string) (*model.Config, error) {
	c.logger.Debug("retrieving notification config from name", zap.String("name", name))

	return c.getByKey(key(name))
}

func (c *client) PersistNotificationConfig(config *model.Config) error {
	c.logger.Debug("persisting notification config", zap.String("name", config.Name))

	jsonStr, err := config.AsJSONString()
	if err != nil {
		c.logger.Error("error marshalling notification config", zap.Error(err), zap.String("name", config.Name))
		return err
	}

	boolCmd := c.rdb.Set(key(config.Name), jsonStr, 0)
	if err := boolCmd.Err(); err != nil {
		c.logger.Error("error persisting notification config", zap.Error(err), zap.String("name", config.Name))
		return err
	}

	return nil
}

func (c *client) getByKey(key string) (*model.Config, error) {
	strCmd := c.rdb.Get(key)
	if err := strCmd.Err(); err != nil {
		return nil, app_errors.LogAndError("error retrieving notification config from key",
			err, c.logger, zap.String("key", key))
	}

	config := &model.Config{}
	if err := config.FromJSONString(strCmd.Val()); err != nil {
		return nil, err
	}

	return config, nil
}

func key(key string) string {
	return fmt.Sprintf("%s:%s", model.NotificationConfigSet, key)
}
