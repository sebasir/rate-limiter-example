package http

import (
	"github.com/gin-gonic/gin"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
	"net/http"
	"time"
)

type Controller interface {
	StartServer() error
	SendNotification(ctx *gin.Context)
}

type controller struct {
	client service.Client
	logger *zap.Logger
}

func NewController(client service.Client) Controller {
	return &controller{
		client: client,
		logger: zap.L(),
	}
}
func (c controller) StartServer() error {
	r := gin.Default()
	r.POST("/send", c.SendNotification)
	return r.Run()
}

func (c controller) SendNotification(ctx *gin.Context) {
	//TODO: parse from request body
	mail := pb.Notification{
		Recipient: "smotavitam@gmail.com",
		Message:   "Hello there, this is our latest news!!!",
	}

	config := pb.Config{
		Name:  "Newsletter",
		Limit: 2,
		Unit:  durationpb.New(time.Second * time.Duration(10)),
	}

	res, err := c.client.Send(&mail, &config)
	if err != nil {
		c.logger.Error("error sending notification to client", zap.Error(err))
		if res == nil {
			ctx.JSON(http.StatusInternalServerError, "empty response message")
			return
		}

		switch res.Status {
		case pb.Status_SENT:
			ctx.JSON(http.StatusOK, "notification sent to recipient, yet an error occurred")
		case pb.Status_REJECTED:
			ctx.JSON(http.StatusTooManyRequests, "notification was rejected by rate limiter, yet an error occurred")
		case pb.Status_ERROR:
			ctx.JSON(http.StatusInternalServerError, "internal server error")
		}

		return
	}

	switch res.Status {
	case pb.Status_SENT:
		ctx.JSON(http.StatusOK, "notification sent to recipient")
	case pb.Status_REJECTED:
		ctx.JSON(http.StatusTooManyRequests, "notification was rejected by rate limiter")
	case pb.Status_ERROR:
		ctx.JSON(http.StatusInternalServerError, "internal server error")
	}
}
