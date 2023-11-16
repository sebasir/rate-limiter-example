package ratelimiter

import (
	"errors"
	"github.com/go-redis/redis"
	. "github.com/ovechkin-dm/mockio/mock"
	"github.com/sebasir/rate-limiter-example/manager"
	"github.com/sebasir/rate-limiter-example/model"
	pb "github.com/sebasir/rate-limiter-example/notification/proto"
	"github.com/sebasir/rate-limiter-example/service"
	"go.uber.org/zap"
	"reflect"
	"testing"
	"time"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
}

type fields struct {
	rdb      redis.Cmdable
	delegate service.Client
	manager  manager.Service
}

type testCase struct {
	name      string
	fields    fields
	args      *pb.Notification
	want      *pb.Result
	wantErr   bool
	targetErr error
}

type redisCmd[T int64 | bool | time.Duration] struct {
	val     T
	err     error
	exclude bool
}

type redisMock struct {
	incrCmd   redisCmd[int64]
	expireCmd redisCmd[bool]
	ttlCmd    redisCmd[time.Duration]
}

func (r *redisMock) buildRedisMock() redis.Cmdable {
	rdbMock := Mock[redis.Cmdable]()

	if !r.incrCmd.exclude {
		When(rdbMock.Incr(AnyString())).
			ThenReturn(redis.NewIntResult(r.incrCmd.val, r.incrCmd.err))
	}

	if !r.ttlCmd.exclude {
		When(rdbMock.TTL(AnyString())).
			ThenReturn(redis.NewDurationResult(r.ttlCmd.val, r.ttlCmd.err))
	}

	if !r.expireCmd.exclude {
		When(rdbMock.Expire(AnyString(), Any[time.Duration]())).
			ThenReturn(redis.NewBoolResult(r.expireCmd.val, r.expireCmd.err))
	}

	return rdbMock
}

type managerMock struct {
	config       *model.Config
	getByNameErr error
}

func (m *managerMock) buildManagerMock() manager.Service {
	mgrMock := Mock[manager.Service]()
	When(mgrMock.GetByName(AnyString())).ThenReturn(m.config, m.getByNameErr)

	return mgrMock
}

type delegateMock struct {
	result  *pb.Result
	sendErr error
}

func (d *delegateMock) buildDelegateMock() service.Client {
	dlgMock := Mock[service.Client]()
	When(dlgMock.Send(Any[*pb.Notification]())).ThenReturn(d.result, d.sendErr)

	return dlgMock
}

func getTestCases() []*testCase {
	okNotification := &pb.Notification{
		Recipient:        "a@a.a",
		Message:          "Hello world",
		NotificationType: "Newsletter",
	}

	okResponse := &pb.Result{
		Status:          pb.Status_SENT,
		ResponseMessage: "notification sent to recipient",
	}

	okConfig := &model.Config{
		Name:       "Newsletter",
		LimitCount: 1,
		TimeAmount: 1,
		TimeUnit:   "MINUTE",
	}

	okMgr := (&managerMock{
		config: okConfig,
	}).buildManagerMock()

	redisErr := errors.New("redis: error")

	return []*testCase{
		{
			name: "OK_Notification_Send",
			fields: fields{
				rdb: (&redisMock{
					incrCmd:   redisCmd[int64]{val: 1},
					expireCmd: redisCmd[bool]{val: true},
					ttlCmd:    redisCmd[time.Duration]{exclude: true},
				}).buildRedisMock(),
				delegate: (&delegateMock{
					result: okResponse,
				}).buildDelegateMock(),
				manager: okMgr,
			},
			args: okNotification,
			want: okResponse,
		}, {
			name: "OK_Notification Rejected",
			fields: fields{
				rdb: (&redisMock{
					incrCmd:   redisCmd[int64]{val: 2},
					expireCmd: redisCmd[bool]{exclude: true},
					ttlCmd:    redisCmd[time.Duration]{val: time.Minute},
				}).buildRedisMock(),
				manager: okMgr,
			},
			args: okNotification,
			want: &pb.Result{
				Status:          pb.Status_REJECTED,
				ResponseMessage: "notification to recipient was rejected",
			},
		}, {
			name: "ERROR_Unknown_Notification_Config",
			fields: fields{
				manager: (&managerMock{
					getByNameErr: redisErr,
				}).buildManagerMock(),
			},
			args:      okNotification,
			want:      InternalErrorResult,
			wantErr:   true,
			targetErr: ErrProcessingNotificationRequest,
		}, {
			name: "ERROR_Redis_Incr",
			fields: fields{
				rdb: (&redisMock{
					incrCmd:   redisCmd[int64]{err: redisErr},
					expireCmd: redisCmd[bool]{exclude: true},
					ttlCmd:    redisCmd[time.Duration]{exclude: true},
				}).buildRedisMock(),
				manager: okMgr,
			},
			args:      okNotification,
			want:      InternalErrorResult,
			wantErr:   true,
			targetErr: ErrProcessingNotificationRequest,
		}, {
			name: "ERROR_Redis_Expire",
			fields: fields{
				rdb: (&redisMock{
					incrCmd:   redisCmd[int64]{val: 1},
					expireCmd: redisCmd[bool]{err: redisErr},
					ttlCmd:    redisCmd[time.Duration]{exclude: true},
				}).buildRedisMock(),
				manager: okMgr,
			},
			args:      okNotification,
			want:      InternalErrorResult,
			wantErr:   true,
			targetErr: ErrProcessingNotificationRequest,
		}, {
			name: "ERROR_Redis_TTL",
			fields: fields{
				rdb: (&redisMock{
					incrCmd:   redisCmd[int64]{val: 2},
					expireCmd: redisCmd[bool]{exclude: true},
					ttlCmd:    redisCmd[time.Duration]{err: redisErr},
				}).buildRedisMock(),
				manager: okMgr,
			},
			args:      okNotification,
			want:      InternalErrorResult,
			wantErr:   true,
			targetErr: ErrProcessingNotificationRequest,
		}, {
			name: "ERROR_Delegate_Send",
			fields: fields{
				rdb: (&redisMock{
					incrCmd:   redisCmd[int64]{val: 1},
					expireCmd: redisCmd[bool]{val: true},
					ttlCmd:    redisCmd[time.Duration]{exclude: true},
				}).buildRedisMock(),
				delegate: (&delegateMock{
					sendErr: errors.New("i/o error"),
				}).buildDelegateMock(),
				manager: okMgr,
			},
			args:      okNotification,
			want:      InternalErrorResult,
			wantErr:   true,
			targetErr: ErrProcessingNotificationRequest,
		},
	}
}

func Test_client_Send(t *testing.T) {
	SetUp(t)
	tests := getTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{
				delegate: tt.fields.delegate,
				manager:  tt.fields.manager,
				rdb:      tt.fields.rdb,
				logger:   zap.L(),
			}
			got, err := c.Send(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !errors.Is(err, tt.targetErr) {
				t.Errorf("Send() error = %v, targetErr = %v", err, tt.targetErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Send() got = %v, want %v", got, tt.want)
			}
		})
	}
}
