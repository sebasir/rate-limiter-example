package http

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type Controller interface {
	StartServer() error
	SendNotification(ctx *gin.Context)
	ListTypes(ctx *gin.Context)
}

type controller struct {
	client       service.Client
	configClient service.ExtendedClient
	logger       *zap.Logger
}

func NewController(client service.Client) Controller {
	return &controller{
		client: client,
		logger: zap.L(),
	}
}

func NewControllerWithConfig(client service.ExtendedClient) Controller {
	return &controller{
		configClient: client,
		client:       service.Client(client),
		logger:       zap.L(),
	}
}

func (c controller) StartServer() error {
	c.logger.Debug("starting GIN server")
	r := gin.Default()
	r.POST("/send", c.SendNotification)
	if c.configClient != nil {
		r.GET("/type/list", c.ListTypes)
	}
	return r.Run()
}

func (c controller) SendNotification(ctx *gin.Context) {
	c.logger.Debug("notification received on GIN handler", zap.String("handler", "SendNotification"))

	jsonData, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		c.logger.Error("error reading request body", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, "error reading request body")
		return
	}

	notification := pb.Notification{}
	if err := json.Unmarshal(jsonData, &notification); err != nil {
		c.logger.Error("error reading request body", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, "error parsing request body")
		return
	}

	c.logger.Debug("notification forwarded to service")
	res, err := c.client.Send(&notification)
	if err != nil {
		c.logger.Error("error sending notification to client", zap.Error(err))
		if res == nil {
			ctx.JSON(http.StatusInternalServerError, "empty response message")
			return
		}

		c.logger.Debug("notification response received (with error)", zap.String("status", res.Status.String()))
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

	c.logger.Debug("notification response received", zap.String("status", res.Status.String()))
	switch res.Status {
	case pb.Status_SENT:
		ctx.JSON(http.StatusOK, "notification sent to recipient")
	case pb.Status_REJECTED:
		ctx.JSON(http.StatusTooManyRequests, "notification was rejected by rate limiter")
	case pb.Status_ERROR:
		ctx.JSON(http.StatusInternalServerError, "internal server error")
	}
}

func (c controller) ListTypes(ctx *gin.Context) {
	configs, err := c.configClient.ListNotificationConfig()
	if err != nil {
		c.logger.Error("error listing notification types", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, "internal server error")
	}

	jsonByte, err := json.Marshal(configs)
	if err != nil {
		c.logger.Error("error marshalling notification types", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, "internal server error")
	}

	ctx.JSON(http.StatusOK, string(jsonByte))
}
