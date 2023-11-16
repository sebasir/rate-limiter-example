package manager

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	. "github.com/sebasir/rate-limiter-example/app_errors"
	"github.com/sebasir/rate-limiter-example/model"
	"go.uber.org/zap"
)

var ErrOperatingNotificationConfig = errors.New("error operating notification config")

type client struct {
	rdb    redis.Cmdable
	logger *zap.Logger
}

func NewClient(rdb redis.Cmdable) Service {
	return &client{
		rdb:    rdb,
		logger: zap.L(),
	}
}

func (c *client) ListNotificationConfig() ([]*model.Config, error) {
	c.logger.Debug("retrieving notification config list")

	scanKey := fmtKey("*")
	var cursor uint64
	allKeys := make([]*model.Config, 0)
	for {
		foundSet, cursor, err := c.rdb.Scan(cursor, scanKey, 50).Result()
		if err != nil {
			return nil, LogAndError("error retrieving notification config list",
				errors.Join(err, ErrOperatingNotificationConfig), c.logger)
		}

		tempConfigs := make([]*model.Config, len(foundSet))
		for i, value := range foundSet {
			config, err := c.getByKey(value)
			if err != nil {
				return nil, LogAndError("error retrieving notification config from fmtKey",
					err, c.logger, zap.String("value", value))
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

	return c.getByKey(fmtKey(name))
}

func (c *client) PersistNotificationConfig(config *model.Config) error {
	configField := zap.String("name", config.Name)
	c.logger.Debug("persisting notification config", configField)

	jsonStr, err := config.AsJSONString()
	if err != nil {
		return LogAndError("error marshalling notification config",
			errors.Join(err, ErrOperatingNotificationConfig), c.logger, configField, configField)
	}

	statusCmd := c.rdb.Set(fmtKey(config.Name), jsonStr, 0)
	if err := statusCmd.Err(); err != nil {
		return LogAndError("error persisting notification config",
			errors.Join(err, ErrOperatingNotificationConfig), c.logger, configField)
	}

	return nil
}

func (c *client) getByKey(key string) (*model.Config, error) {
	keyField := zap.String("key", key)
	strCmd := c.rdb.Get(key)
	if err := strCmd.Err(); err != nil {
		return nil, LogAndError("error retrieving notification config from fmtKey",
			errors.Join(err, ErrOperatingNotificationConfig), c.logger, keyField)
	}

	config := &model.Config{}
	if err := config.FromJSONString(strCmd.Val()); err != nil {
		return nil, LogAndError("error parsing notification config from DB",
			errors.Join(err, ErrOperatingNotificationConfig), c.logger, keyField)
	}

	return config, nil
}

func fmtKey(key string) string {
	return fmt.Sprintf("%s:%s", model.NotificationConfigSet, key)
}
