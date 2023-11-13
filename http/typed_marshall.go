package http

import (
	"encoding/json"
	"errors"
	"github.com/sebasir/rate-limiter-example/model"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"io"
)

var (
	errReadingRequestBody = errors.New("error parsing request body")
	errParsingRequestBody = errors.New("error reading request body")
)

func ParseRequestBody[V pb.Notification | model.Config](r io.Reader) (*V, error) {
	jsonData, err := io.ReadAll(r)
	if err != nil {
		return nil, errReadingRequestBody
	}

	target := new(V)
	if err := json.Unmarshal(jsonData, target); err != nil {
		return nil, errParsingRequestBody
	}

	return target, nil
}
