package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	. "github.com/ovechkin-dm/mockio/mock"
	"github.com/sebasir/rate-limiter-example/model"
	"github.com/sebasir/rate-limiter-example/notification/proto"
	ratelimiter "github.com/sebasir/rate-limiter-example/rate_limiter"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
}

var (
	configStrMap = map[string]string{
		"News":      `{"name":"News","limitCount":1,"timeUnit":"DAY","timeAmount":1}`,
		"Status":    `{"name":"Status","limitCount":2,"timeUnit":"MINUTE","timeAmount":1}`,
		"Marketing": `{"name":"Marketing","limitCount":3,"timeUnit":"HOUR","timeAmount":1}`,
	}

	configMap = map[string]*model.Config{
		"News": {
			Name:       "News",
			LimitCount: 1,
			TimeAmount: 1,
			TimeUnit:   "DAY",
		},
		"Status": {
			Name:       "Status",
			LimitCount: 2,
			TimeAmount: 1,
			TimeUnit:   "MINUTE",
		},
		"Marketing": {
			Name:       "Marketing",
			LimitCount: 3,
			TimeAmount: 1,
			TimeUnit:   "HOUR",
		},
	}

	val = GetValidator()

	backendErr = errors.New("some backend error")
)

type testCase struct {
	name          string
	fields        fields
	wantedStatus  int
	wantedMessage string
	wantedList    []*model.Config
}

type fields struct {
	client       service.Client
	configClient service.ExtendedClient
	input        string
}

func getTestGinContext(w *httptest.ResponseRecorder) *gin.Context {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}

	return ctx
}

type extendedClientMock struct {
	persistConfigErr     error
	persistConfigExclude bool
	ListConfigsVal       []*model.Config
	ListConfigsErr       error
	ListConfigsExclude   bool
}

func (c *extendedClientMock) buildMock() service.ExtendedClient {
	extClientMock := Mock[service.ExtendedClient]()

	if !c.persistConfigExclude {
		When(extClientMock.PersistNotificationConfig(Any[*model.Config]())).
			ThenReturn(c.persistConfigErr)
	}

	if !c.ListConfigsExclude {
		When(extClientMock.ListNotificationConfig()).
			ThenReturn(c.ListConfigsVal, c.ListConfigsErr)
	}

	return extClientMock
}

type clientMock struct {
	sendErr     error
	sendVal     *proto.Result
	sendExclude bool
}

func (c *clientMock) buildMock() service.Client {
	mock := Mock[service.Client]()

	if !c.sendExclude {
		When(mock.Send(Any[*proto.Notification]())).
			ThenReturn(c.sendVal, c.sendErr)
	}

	return mock
}

func Test_controller_ListNotificationTypes(t *testing.T) {
	SetUp(t)
	mockGet := func(c *gin.Context) {
		c.Request.Method = "GET"
		c.Request.Header.Set("Content-Type", "application/json")
	}

	okConfigs := make([]*model.Config, 0, len(configMap))
	for _, value := range configMap {
		okConfigs = append(okConfigs, value)
	}

	tests := []testCase{
		{
			name: "OK_Notification_Config_Listed",
			fields: fields{
				configClient: (&extendedClientMock{
					persistConfigExclude: true,
					ListConfigsVal:       okConfigs,
				}).buildMock(),
			},
			wantedStatus: http.StatusOK,
			wantedList:   okConfigs,
		}, {
			name: "ERROR_No_Notification_Configs_Found",
			fields: fields{
				configClient: (&extendedClientMock{
					persistConfigExclude: true,
					ListConfigsVal:       []*model.Config{},
				}).buildMock(),
			},
			wantedStatus: http.StatusNoContent,
		}, {
			name: "ERROR_Listing_Configs_From_Backend",
			fields: fields{
				configClient: (&extendedClientMock{
					persistConfigExclude: true,
					ListConfigsErr:       backendErr,
				}).buildMock(),
			},
			wantedStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := controller{
				client:       tt.fields.client,
				configClient: tt.fields.configClient,
				logger:       zap.L(),
				validator:    val,
			}

			w := httptest.NewRecorder()
			ctx := getTestGinContext(w)
			mockGet(ctx)

			c.ListNotificationTypes(getTestGinContext(w))

			assert.Equal(t, tt.wantedStatus, w.Code)

			if len(tt.wantedMessage) > 0 {
				assert.Equal(t, tt.wantedMessage, w.Body.String())
			}

			if len(tt.wantedList) > 0 {
				var res []*model.Config

				if err := json.Unmarshal([]byte(w.Body.String()), &res); err != nil {
					t.Errorf("error unmarshalling response: %v", err)
					return
				}

				for _, out := range res {
					if in, exist := configMap[out.Name]; exist {
						assert.Equal(t, out, in)
						continue
					}
					t.Errorf("unexpected response %v, should exist from input", out)
				}
			}
		})
	}
}

func Test_controller_SaveNotificationType(t *testing.T) {
	SetUp(t)
	mockPut := func(c *gin.Context, content string) {
		c.Request.Method = "PUT"
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
	}

	tests := []testCase{
		{
			name: "OK_Notification_Config_Saved",
			fields: fields{
				configClient: (&extendedClientMock{
					ListConfigsExclude: true,
				}).buildMock(),
				input: configStrMap["News"],
			},
			wantedStatus: http.StatusNoContent,
		}, {
			name: "VALIDATION_Invalid_Input",
			fields: fields{
				configClient: (&extendedClientMock{
					persistConfigExclude: true,
					ListConfigsExclude:   true,
				}).buildMock(),
				input: `{"hello":"world"}`,
			},
			wantedStatus:  http.StatusBadRequest,
			wantedMessage: `{"error":{"Config.LimitCount":"LimitCount must be 1 or greater","Config.Name":"Name is a required field","Config.TimeAmount":"TimeAmount must be 1 or greater","Config.TimeUnit":"TimeUnit must be one of SECOND, MINUTE, HOUR, DAY"},"message":"error processing input"}`,
		}, {
			name: "VALIDATION_Missing_Required_Field_Input",
			fields: fields{
				configClient: (&extendedClientMock{
					persistConfigExclude: true,
					ListConfigsExclude:   true,
				}).buildMock(),
				input: `{"limitCount":1, "timeUnit":"DAY","timeAmount":1}`,
			},
			wantedStatus:  http.StatusBadRequest,
			wantedMessage: `{"error":{"Config.Name":"Name is a required field"},"message":"error processing input"}`,
		}, {
			name: "ERROR_Error_Persisting",
			fields: fields{
				configClient: (&extendedClientMock{
					persistConfigErr:   backendErr,
					ListConfigsExclude: true,
				}).buildMock(),
				input: configStrMap["Status"],
			},
			wantedStatus:  http.StatusBadRequest,
			wantedMessage: `{"error":"some backend error","message":"error persisting notification input"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := controller{
				client:       tt.fields.client,
				configClient: tt.fields.configClient,
				logger:       zap.L(),
				validator:    val,
			}

			w := httptest.NewRecorder()
			ctx := getTestGinContext(w)
			mockPut(ctx, tt.fields.input)
			c.SaveNotificationType(ctx)

			assert.Equal(t, tt.wantedStatus, w.Code)
			assert.Equal(t, tt.wantedMessage, w.Body.String())
		})
	}
}

func Test_controller_SendNotification(t *testing.T) {
	SetUp(t)
	mockPost := func(c *gin.Context, content string) {
		c.Request.Method = "POST"
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
	}

	okNotification := `{"notificationType":"Newsletter","recipient":"smotavitam@gmail.com","message":"Our latest news: Hi!"}`

	notificationSent := &proto.Result{
		Status:          proto.Status_SENT,
		ResponseMessage: "notification sent to recipient",
	}

	notificationRejected := &proto.Result{
		Status:          proto.Status_REJECTED,
		ResponseMessage: "notification to recipient was rejected",
	}

	invalidNotification := &proto.Result{
		Status:          proto.Status_INVALID_NOTIFICATION,
		ResponseMessage: "invalid notification on backend server",
	}

	tests := []testCase{
		{
			name: "OK_Notification_Sent",
			fields: fields{
				client: (&clientMock{
					sendVal: notificationSent,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusOK,
			wantedMessage: `{"message":"notification sent to recipient"}`,
		}, {
			name: "OK_Notification_Rejected",
			fields: fields{
				client: (&clientMock{
					sendVal: notificationRejected,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusTooManyRequests,
			wantedMessage: `{"message":"notification was rejected by rate limiter"}`,
		}, {
			name: "NO_ERROR_Invalid_Input",
			fields: fields{
				client: (&clientMock{
					sendVal: invalidNotification,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusBadRequest,
			wantedMessage: `{"message":"unrecognized input"}`,
		}, {
			name: "NO_ERROR_Internal_Server_Error",
			fields: fields{
				client: (&clientMock{
					sendVal: ratelimiter.InternalErrorResult,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusInternalServerError,
			wantedMessage: `{"message":"internal server error"}`,
		}, {
			name: "VALIDATION_Invalid_Input",
			fields: fields{
				client: (&clientMock{
					sendExclude: true,
				}).buildMock(),
				input: `{"hello":"world"}`,
			},
			wantedStatus:  http.StatusBadRequest,
			wantedMessage: `{"error":{"Notification.Message":"Message is a required field","Notification.NotificationType":"NotificationType is a required field","Notification.Recipient":"Recipient is a required field"},"message":"error processing input"}`,
		}, {
			name: "VALIDATION_Missing_Required_Field_Input",
			fields: fields{
				client: (&clientMock{
					sendExclude: true,
				}).buildMock(),
				input: `{"notificationType":"Newsletter","message":"Our latest news: Hi!"}`,
			},
			wantedStatus:  http.StatusBadRequest,
			wantedMessage: `{"error":{"Notification.Recipient":"Recipient is a required field"},"message":"error processing input"}`,
		}, {
			name: "ERROR_Error_Sending_Notification",
			fields: fields{
				client: (&clientMock{
					sendVal: ratelimiter.InternalErrorResult,
					sendErr: backendErr,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusInternalServerError,
			wantedMessage: `{"error":"some backend error","message":"error occurred while processing request"}`,
		}, {
			name: "ERROR_Error_Sending_Notification_Empty",
			fields: fields{
				client: (&clientMock{
					sendErr: backendErr,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusInternalServerError,
			wantedMessage: `{"error":"some backend error","message":"empty response message"}`,
		}, {
			name: "ERROR_Error_Notification_Sent",
			fields: fields{
				client: (&clientMock{
					sendErr: backendErr,
					sendVal: notificationSent,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusOK,
			wantedMessage: `{"error":"some backend error","message":"notification sent to recipient, yet an error occurred"}`,
		}, {
			name: "ERROR_Error_Notification_Rejected",
			fields: fields{
				client: (&clientMock{
					sendErr: backendErr,
					sendVal: notificationRejected,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusTooManyRequests,
			wantedMessage: `{"error":"some backend error","message":"notification was rejected by rate limiter, yet an error occurred"}`,
		}, {
			name: "ERROR_Error_Internal_Server_Error",
			fields: fields{
				client: (&clientMock{
					sendErr: backendErr,
					sendVal: ratelimiter.InternalErrorResult,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusInternalServerError,
			wantedMessage: `{"error":"some backend error","message":"error occurred while processing request"}`,
		}, {
			name: "ERROR_Error_Invalid_Input",
			fields: fields{
				client: (&clientMock{
					sendErr: backendErr,
					sendVal: invalidNotification,
				}).buildMock(),
				input: okNotification,
			},
			wantedStatus:  http.StatusBadRequest,
			wantedMessage: `{"error":"some backend error","message":"invalid user input"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := controller{
				client:       tt.fields.client,
				configClient: tt.fields.configClient,
				logger:       zap.L(),
				validator:    val,
			}

			w := httptest.NewRecorder()
			ctx := getTestGinContext(w)
			mockPost(ctx, tt.fields.input)
			c.SendNotification(ctx)

			assert.Equal(t, tt.wantedStatus, w.Code)
			assert.Equal(t, tt.wantedMessage, w.Body.String())
		})
	}
}
