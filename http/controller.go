package http

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/sebasir/rate-limiter-example/model"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
	"net/http"
)

type Controller interface {
	StartServer() error
	SendNotification(ctx *gin.Context)
	ListNotificationTypes(ctx *gin.Context)
	SaveNotificationType(ctx *gin.Context)
}

type controller struct {
	client       service.Client
	configClient service.ExtendedClient
	logger       *zap.Logger
	validator    *Validator
}

func NewController(client service.Client) Controller {
	return &controller{
		client:    client,
		logger:    zap.L(),
		validator: GetValidator(),
	}
}

func NewControllerWithConfig(client service.ExtendedClient) Controller {
	return &controller{
		configClient: client,
		client:       service.Client(client),
		logger:       zap.L(),
		validator:    GetValidator(),
	}
}

func (c controller) StartServer() error {
	c.logger.Debug("starting GIN server")
	r := gin.Default()
	r.POST("/send", c.SendNotification)
	if c.configClient != nil {
		r.GET("/type/list", c.ListNotificationTypes)
		r.PUT("/type/", c.SaveNotificationType)
	}
	return r.Run()
}

func (c controller) SendNotification(ctx *gin.Context) {
	c.logger.Debug("notification received on GIN handler", zap.String("handler", "SendNotification"))

	notification, err := ParseRequestBody[pb.Notification](ctx.Request.Body)
	if err != nil {
		c.logger.Error("error parsing request body", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "error processing input",
			"error":   err,
		})
		return
	}

	if err := c.validator.Struct(notification); err != nil {
		c.logger.Error("error parsing request body", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "error processing input",
			"error":   c.validator.Translate(err),
		})
		return
	}

	c.logger.Debug("notification forwarded to service")
	res, err := c.client.Send(notification)
	if err != nil {
		c.logger.Error("error sending notification to client", zap.Error(err))
		if res == nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": "empty response message",
				"error":   err,
			})
			return
		}

		c.logger.Debug("notification response received (with error)", zap.String("status", res.Status.String()))
		switch res.Status {
		case pb.Status_SENT:
			ctx.JSON(http.StatusOK, gin.H{
				"message": "notification sent to recipient, yet an error occurred",
				"error":   err,
			})
		case pb.Status_REJECTED:
			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"message": "notification was rejected by rate limiter, yet an error occurred",
				"error":   err,
			})
		case pb.Status_INTERNAL_ERROR:
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": "error occurred while processing request",
				"error":   err.Error(),
			})
		case pb.Status_INVALID_NOTIFICATION:
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid user input",
				"error":   err,
			})
		}

		return
	}

	c.logger.Debug("notification response received", zap.String("status", res.Status.String()))
	switch res.Status {
	case pb.Status_SENT:
		ctx.JSON(http.StatusOK, gin.H{
			"message": "notification sent to recipient",
		})
	case pb.Status_REJECTED:
		ctx.JSON(http.StatusTooManyRequests, gin.H{
			"message": "notification was rejected by rate limiter",
		})
	case pb.Status_INTERNAL_ERROR:
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
		})
	case pb.Status_INVALID_NOTIFICATION:
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "unrecognized input",
		})
	}
}

func (c controller) ListNotificationTypes(ctx *gin.Context) {
	configs, err := c.configClient.ListNotificationConfig()
	if err != nil {
		c.logger.Error("error listing notification types", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
			"error":   err,
		})
		return
	}

	if len(configs) == 0 {
		c.logger.Info("no notification types found")
		ctx.JSON(http.StatusNoContent, gin.H{
			"message": "no notification types found",
		})
		return
	}

	jsonByte, err := json.Marshal(configs)
	if err != nil {
		c.logger.Error("error marshalling notification types", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
			"error":   err,
		})
		return
	}

	ctx.JSON(http.StatusOK, string(jsonByte))
}

func (c controller) SaveNotificationType(ctx *gin.Context) {
	config, err := ParseRequestBody[model.Config](ctx.Request.Body)
	if err != nil {
		c.logger.Error("error parsing request body", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "error processing input",
			"error":   err,
		})
		return
	}

	if err := c.validator.Struct(config); err != nil {
		c.logger.Error("error parsing request body", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "error processing input",
			"error":   err,
		})
		return
	}

	if err := c.configClient.PersistNotificationConfig(config); err != nil {
		c.logger.Error("error persisting notification config", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "error persisting notification input",
			"error":   err,
		})
	}
}
