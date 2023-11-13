package model

import (
	"encoding/json"
	"errors"
	"time"
)

const NotificationConfigSet = "NOTIFICATION_CONFIG"

var (
	ErrUnknownTimeUnitInput = errors.New("unknown input for time unit field")
)

type Config struct {
	Name       string        `json:"name" validate:"required"`
	LimitCount int64         `json:"limitCount" validate:"gte=1"`
	TimeAmount int64         `json:"timeAmount" validate:"gte=1"`
	TimeUnit   time.Duration `json:"timeUnit" validate:"required"`
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

func (c *Config) UnmarshalJSON(data []byte) error {
	timeUnitMap := map[string]time.Duration{
		"SECOND": time.Second,
		"MINUTE": time.Minute,
		"HOUR":   time.Hour,
		"DAY":    time.Hour * time.Duration(24),
	}

	type Alias Config
	aux := &struct {
		TimeUnit string `json:"timeUnit"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if duration, exist := timeUnitMap[aux.TimeUnit]; exist {
		c.TimeUnit = duration
		return nil
	}

	return ErrUnknownTimeUnitInput
}

func (c *Config) MarshalJSON() ([]byte, error) {
	timeUnitMap := map[time.Duration]string{
		time.Second:                   "SECOND",
		time.Minute:                   "MINUTE",
		time.Hour:                     "HOUR",
		time.Hour * time.Duration(24): "DAY",
	}

	type Alias Config
	aux := &struct {
		TimeUnit string `json:"timeUnit"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	aux.TimeUnit = timeUnitMap[c.TimeUnit]
	data, err := json.Marshal(&aux)
	if err != nil {
		return nil, err
	}

	return data, nil
}
