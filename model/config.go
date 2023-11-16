package model

import (
	"encoding/json"
	"time"
)

const NotificationConfigSet = "NOTIFICATION_CONFIG"

var (
	TimeUnitMap = map[string]time.Duration{
		"SECOND": time.Second,
		"MINUTE": time.Minute,
		"HOUR":   time.Hour,
		"DAY":    time.Hour * time.Duration(24),
	}
)

type Config struct {
	Name       string `json:"name" validate:"required"`
	LimitCount int64  `json:"limitCount" validate:"gte=1"`
	TimeAmount int64  `json:"timeAmount" validate:"gte=1"`
	TimeUnit   string `json:"timeUnit" validate:"time-unit"`
}

func (c *Config) AsJSONString() (string, error) {
	str, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(str), nil
}

func (c *Config) FromJSONString(raw string) error {
	if err := json.Unmarshal([]byte(raw), c); err != nil {
		return err
	}

	return nil
}

func (c *Config) CalculateTime() time.Duration {
	return TimeUnitMap[c.TimeUnit] * time.Duration(c.TimeAmount)
}
