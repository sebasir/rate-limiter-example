package manager

import (
	"errors"
	"github.com/go-redis/redis"
	. "github.com/ovechkin-dm/mockio/mock"
	"github.com/sebasir/rate-limiter-example/model"
	"go.uber.org/zap"
	"reflect"
	"testing"
	"time"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
}

type testCase[T any, V any] struct {
	name      string
	rdb       redis.Cmdable
	args      V
	want      T
	wantErr   bool
	targetErr error
}

type redisCmd[T scanRes | string | strings] struct {
	val     T
	err     error
	exclude bool
}

type strings []string

type scanRes struct {
	keys   []string
	cursor uint64
}

var redisErr = errors.New("redis: error")

var (
	configStrMap = map[string]string{
		fmtKey("News"):      `{"name":"News","limitCount":1,"timeUnit":"DAY","timeAmount":1}`,
		fmtKey("Status"):    `{"name":"Status","limitCount":2,"timeUnit":"MINUTE","timeAmount":1}`,
		fmtKey("Marketing"): `{"name":"Marketing","limitCount":3,"timeUnit":"HOUR","timeAmount":1}`,
	}

	configMap = map[string]*model.Config{
		fmtKey("News"): {
			Name:       "News",
			LimitCount: 1,
			TimeAmount: 1,
			TimeUnit:   "DAY",
		},
		fmtKey("Status"): {
			Name:       "Status",
			LimitCount: 2,
			TimeAmount: 1,
			TimeUnit:   "MINUTE",
		},
		fmtKey("Marketing"): {
			Name:       "Marketing",
			LimitCount: 3,
			TimeAmount: 1,
			TimeUnit:   "HOUR",
		},
	}
)

type redisMock struct {
	scanCmd redisCmd[scanRes]
	getCmd  redisCmd[strings]
	setCmd  redisCmd[string]
}

func (r *redisMock) buildRedisMock() redis.Cmdable {
	rdbMock := Mock[redis.Cmdable]()

	if !r.scanCmd.exclude {
		When(rdbMock.Scan(Any[uint64](), AnyString(), Any[int64]())).
			ThenReturn(redis.NewScanCmdResult(r.scanCmd.val.keys, r.scanCmd.val.cursor, r.scanCmd.err))
	}

	if !r.getCmd.exclude {
		applier := func(key string) {
			When(rdbMock.Get(key)).
				ThenReturn(redis.NewStringResult(configStrMap[key], r.getCmd.err))
		}

		if len(r.getCmd.val) == 0 && r.getCmd.err != nil {
			When(rdbMock.Get(AnyString())).
				ThenReturn(redis.NewStringResult("{}", r.getCmd.err))
		}

		for _, key := range r.getCmd.val {
			applier(fmtKey(key))
		}
	}

	if !r.setCmd.exclude {
		When(rdbMock.Set(AnyString(), AnyInterface(), Any[time.Duration]())).
			ThenReturn(redis.NewStatusResult(r.setCmd.val, r.setCmd.err))
	}

	return rdbMock
}

func Test_client_GetByName(t *testing.T) {
	SetUp(t)

	tests := []testCase[*model.Config, string]{
		{
			name: "OK_Config_Retrieved",
			rdb: (&redisMock{
				getCmd:  redisCmd[strings]{val: strings{"News"}},
				setCmd:  redisCmd[string]{exclude: true},
				scanCmd: redisCmd[scanRes]{exclude: true},
			}).buildRedisMock(),
			args: "News",
			want: configMap[fmtKey("News")],
		}, {
			name: "ERROR_Redis_Get",
			rdb: (&redisMock{
				getCmd:  redisCmd[strings]{err: redisErr},
				setCmd:  redisCmd[string]{exclude: true},
				scanCmd: redisCmd[scanRes]{exclude: true},
			}).buildRedisMock(),
			args:      "News",
			wantErr:   true,
			targetErr: ErrOperatingNotificationConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{
				rdb:    tt.rdb,
				logger: zap.L(),
			}
			got, err := c.GetByName(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !errors.Is(err, tt.targetErr) {
				t.Errorf("GetByName() error = %v, targetErr = %v", err, tt.targetErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetByName() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_ListNotificationConfig(t *testing.T) {
	SetUp(t)

	jsonStrConfigs := make(strings, 0, len(configStrMap))
	jsonStrKeys := make(strings, 0, len(configStrMap))
	expectedConfigs := make([]*model.Config, 0, len(configMap))
	for key, value := range configStrMap {
		jsonStrKeys = append(jsonStrKeys, key)
		jsonStrConfigs = append(jsonStrConfigs, value)
		expectedConfigs = append(expectedConfigs, configMap[key])
	}

	tests := []testCase[[]*model.Config, string]{
		{
			name: "OK_Configs_Retrieved",
			rdb: (&redisMock{
				getCmd: redisCmd[strings]{val: strings{"Marketing", "News", "Status"}},
				setCmd: redisCmd[string]{exclude: true},
				scanCmd: redisCmd[scanRes]{
					val: scanRes{
						keys:   jsonStrKeys,
						cursor: 0,
					}},
			}).buildRedisMock(),
			want: expectedConfigs,
		}, {
			name: "OK_No_Configs_Retrieved",
			rdb: (&redisMock{
				getCmd: redisCmd[strings]{val: strings{}},
				setCmd: redisCmd[string]{exclude: true},
				scanCmd: redisCmd[scanRes]{
					val: scanRes{
						keys:   strings{},
						cursor: 0,
					}},
			}).buildRedisMock(),
			want: []*model.Config{},
		}, {
			name: "ERROR_Redis_Get",
			rdb: (&redisMock{
				getCmd: redisCmd[strings]{err: redisErr},
				setCmd: redisCmd[string]{exclude: true},
				scanCmd: redisCmd[scanRes]{
					val: scanRes{
						keys:   strings{"Marketing"},
						cursor: 0,
					}},
			}).buildRedisMock(),
			wantErr:   true,
			targetErr: ErrOperatingNotificationConfig,
		}, {
			name: "ERROR_Redis_Scan",
			rdb: (&redisMock{
				getCmd:  redisCmd[strings]{exclude: true},
				setCmd:  redisCmd[string]{exclude: true},
				scanCmd: redisCmd[scanRes]{err: redisErr},
			}).buildRedisMock(),
			wantErr:   true,
			targetErr: ErrOperatingNotificationConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{
				rdb:    tt.rdb,
				logger: zap.L(),
			}
			got, err := c.ListNotificationConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListNotificationConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !errors.Is(err, tt.targetErr) {
				t.Errorf("ListNotificationConfig() error = %v, targetErr = %v", err, tt.targetErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListNotificationConfig() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_client_PersistNotificationConfig(t *testing.T) {
	SetUp(t)

	tests := []testCase[any, *model.Config]{
		{
			name: "OK_Config_Persisted",
			rdb: (&redisMock{
				getCmd:  redisCmd[strings]{exclude: true},
				setCmd:  redisCmd[string]{val: "redis: ok"},
				scanCmd: redisCmd[scanRes]{exclude: true},
			}).buildRedisMock(),
			args: configMap[fmtKey("News")],
		}, {
			name: "ERROR_Redis_Set",
			rdb: (&redisMock{
				getCmd:  redisCmd[strings]{exclude: true},
				setCmd:  redisCmd[string]{err: redisErr},
				scanCmd: redisCmd[scanRes]{exclude: true},
			}).buildRedisMock(),
			args:      configMap[fmtKey("News")],
			wantErr:   true,
			targetErr: ErrOperatingNotificationConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{
				rdb:    tt.rdb,
				logger: zap.L(),
			}

			err := c.PersistNotificationConfig(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("PersistNotificationConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, tt.targetErr) {
				t.Errorf("PersistNotificationConfig() error = %v, targetErr = %v", err, tt.targetErr)
				return
			}
		})
	}
}
