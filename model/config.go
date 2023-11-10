package model

import (
	"encoding/json"
	"time"
)

const NotificationConfigSet = "NOTIFICATION_CONFIG"

type Config struct {
	Name  string        `json:"name"`
	Limit int64         `json:"limit"`
	Unit  time.Duration `json:"time_unit"`
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
