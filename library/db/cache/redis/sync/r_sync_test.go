package sync

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	consistent_hash2 "github.com/coderyw/gomicro-util/library/consistent_hash"
	"github.com/coderyw/gomicro-util/library/cron"
	redis2 "github.com/coderyw/gomicro-util/library/db/cache/redis/facade"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/model"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/prefixenum"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/single"
	"github.com/coderyw/gomicro-util/mocks/library/db/cache/redis/facade"
	"github.com/coderyw/gomicro-util/mocks/model/consistent_hash"
	consistent_hash3 "github.com/coderyw/gomicro-util/model/consistent_hash"
	"github.com/coderyw/gomicro-util/model/cst"
	"go.uber.org/zap"
)

func TestRedisSyncMgr_AddJob(t *testing.T) {
	// NS→SAAS 合并说明：本用例是针对 SAAS 旧版 ZSet 引擎的手动观察 harness，
	// 结尾 `<-sig` 会阻塞等待 OS kill 信号（并非自动化断言），且 mock 的
	// ZRangeByScoreH/ZRemRangeByScore 已随引擎替换为 NS 的 version-based Lua 而不再被调用。
	// 新引擎的自动化覆盖见 r_sync_lua_test.go（含丢数据修复 RemSyncItem_VersionMismatch_SkipsZREM 等）。
	// 保留文件以备人工联调，CI 中跳过避免永久阻塞。
	t.Skip("manual harness for legacy engine; blocks on OS signal — superseded by r_sync_lua_test.go")
	// 该用例会在 <-sig 处阻塞等待进程收到 kill 信号，仅适合手动观察运行，
	// 常规 go test 下会导致整包超时失败，故默认跳过
	if os.Getenv("MANUAL_TEST") == "" {
		t.Skip("阻塞等待 OS 信号的手动用例，设置 MANUAL_TEST=1 后运行")
	}
	os.Setenv(cst.ENV_ID, "1")

	sig := single.OsKillSingle()
	type args struct {
		ctx    context.Context
		opt    model.RedisSyncOption
		client redis2.RedisInterface
	}
	tests := []struct {
		name     string
		wantErr  bool
		initMock func(ctx context.Context, t2 *testing.T) args
	}{
		{name: "success add and no sync", wantErr: false, initMock: func(ctx context.Context, t2 *testing.T) args {
			// 初始定时器
			cron.Init()
			chl := consistent_hash.NewMockListener(t2)
			// 一致性哈希初始化
			changeNotice := make(chan consistent_hash3.Notice)
			go func() {
				for {
					time.Sleep(69 * time.Second)
					//随机生成服务
					n := rand.Int63n(10) + 1
					var arr []consistent_hash3.Instance
					for i := range n {
						arr = append(arr, consistent_hash3.Instance{
							Id:   fmt.Sprintf("%d", i),
							Host: fmt.Sprintf("127.0.0.1:%d", i),
						})
					}

					changeNotice <- consistent_hash3.Notice{
						ServiceName: "test",
						Services:    arr,
					}
				}
			}()
			chl.On("OnChanged").Return(changeNotice).Once()
			chl.On("OnNow").Return(consistent_hash3.Notice{
				ServiceName: "test-add-sync",
				Err:         nil,
				Services: []consistent_hash3.Instance{
					consistent_hash3.Instance{
						Host: "127.0.0.1",
						Id:   "1",
					},
					consistent_hash3.Instance{
						Host: "127.0.0.1",
						Id:   "2",
					},
				},
			}).Once()
			consistent_hash2.Init(chl)
			redisInterface := facade.NewMockRedisInterface(t2)
			redisInterface.On("ZRangeByScoreH", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(func(ctx context.Context, prefix prefixenum.PrefixEnum, key string, min, max string, offset, limit int64) ([]string, error) {
					data := waitSyncData[key]
					if len(data) == 0 {
						return nil, nil
					}
					if offset > int64(len(data)) {
						return nil, nil
					}
					if offset+limit > int64(len(data)) {
						limit = int64(len(data)) - offset
					}
					return data[offset : offset+limit], nil
				}).Maybe()
			//Return([]string{"test-add-sync-1", "test-add-sync-2"}, nil)
			redisInterface.On("ZRemRangeByScore", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(int64(2), nil).Maybe()

			return args{
				ctx: metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok"),
				opt: model.RedisSyncOption{
					SyncKey: "test-add-sync",
					Ts:      10,
					DoSync: func(ctx context.Context, s string) error {
						log.InfoCtx(ctx, "开始同步", zap.Any("key", s))
						return nil
					},
					QueueNum: 20,
				},
				client: redisInterface,
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arg := tt.initMock(context.Background(), t)
			r, err := NewRedisSyncMgr()
			if (err != nil) != tt.wantErr {
				t.Errorf("AddJob() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := r.AddJob(arg.ctx, arg.opt, arg.client); (err != nil) != tt.wantErr {
				t.Errorf("AddJob() error = %v, wantErr %v", err, tt.wantErr)
			}
			<-sig
			cron.CloseCron()
		})
	}

}
