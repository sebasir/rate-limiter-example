package manager

import (
	"github.com/sebasir/rate-limiter-example/model"
	"github.com/sebasir/rate-limiter-example/service"
)

type Service interface {
	service.ConfigClient
	GetByName(name string) (*model.Config, error)
}
