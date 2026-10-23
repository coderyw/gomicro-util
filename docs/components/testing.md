# 单元测试与 mock

> 返回 [README](../../README.md)。

## mocks 目录

仓库根目录 `mocks/` 已为 common-util 中的 interface 生成 mock 代码（基于 testify/mockery），目录层级与源码一致，例如 `mocks/library/mq/facade`。

## library/mock

mock 测试组件，通过 context 进行上下文的开启处理

- 开启 mock

  ```go
    // MockSetUp 开启 mock，仅在开发和测试环境可以使用
    func MockSetUp(ctx context.Context, t *testing.T) context.Context
  ```

- 校验并获取 mock 客户端

  ```go
    // 由于有些 mock 接口是单例模式。所以需要对当前 mock 协程使用同一个 mock 客户端。
    // 这个方法会提供验证 mock 是否开启，以及开启后，通过初始化函数生成当前 context 下的唯一 mock 客户端。
    CheckAndGetMockClient[T any](ctx context.Context, f func(t *testing.T) T) T
    // 返回的 T 非 nil 时，表示 mock 已开启
  ```

- mock 示例

  - 单测的方法

    ```go
      // GetKYCCompleteAct 获取完成 kyc 派彩活动配置以及状态
      //
      // Parameters:
      //   - ctx context.Context 上线文
      //   - userId int64 用户 id
      //
      // Returns：
      //   - activity_proto.KYC_ACTIVITY_STATUS 活动状态
      //   - string 活动奖励金额
      //   - activity_proto.KYC_REWARD_STATUS 用户完成状态
      //   - error 错误信息
      func (a *activityRegister) GetKYCCompleteAct(ctx context.Context) (activity_proto.KYC_ACTIVITY_STATUS, string, activity_proto.KYC_REWARD_STATUS, error)
    ```

    这个方法需要调用两个接口：

    - 一个是`registerRepo.GetStatusAndBonusInCache` 获取缓存中的数据的 Dao 层接口
    - 一个是`ca_user_api.CaUserClient(ctx).GetExtendInfoList` 通过 User 微服务获取用户扩展信息的微服务接口

  - 单测 mock 用例示例

    ```go
      func Test_activityRegister_GetKYCCompleteAct(t *testing.T) {
        type fields struct {
          registerRepo dao2.ActivityRegisterBonusRepo
        }
        type args struct {
          ctx context.Context
        }
        // 定义测试用例参数
        tests := []struct {
          name                 string
          // 测试函数参数
          args                 args
          // 测试函数所属结构体参数
          fields               fields
          // 期望的 接口第一个返回值，活动状态
          wantActStatus        activity_proto.KYC_ACTIVITY_STATUS
          // 期望的 接口第二个返回值，活动奖励金额
          wantActReward        string
          // 期望的 接口第三个返回值，用户领奖状态
          wantUserRewardStatus activity_proto.KYC_REWARD_STATUS
          // 期望的 接口第四个返回值，是否需要错误
          wantErr              bool
          // mock 初始化函数
          mockInit             func(ctx context.Context, t *testing.T) (context.Context, fields)
        }{
          {
            name: "活动状态=1,活动奖金=12,玩家活动完成状态=1",
            args: args{
              ctx: trace.GetTraceCtx(metadata.SetCtxData(context.TODO(), metadata.CONTEXT_USER_ID, int64(1234))),
            },
            // 期望 活动状态为打开
            wantActStatus:        activity_proto.KYC_ACTIVITY_STATUS_ON,
            // 期望 活动奖金为 12
            wantActReward:        "12",
            // 期望 用户奖金状态为完成
            wantUserRewardStatus: activity_proto.KYC_REWARD_STATUS_COMPLETED,
            // 期望 测试函数不返回 error
            wantErr:              false,
            mockInit: func(ctx context.Context, t *testing.T) (context.Context, fields) {
              // 开启 mock
              ctx = mock.MockSetUp(ctx, t)
              //  初始化 registerRepo mock 客户端
              regBonusRepo := dao.NewMockActivityRegisterBonusRepo(t)
              // 对 GetStatusAndBonusInCache 方法打桩，并返回测试数据，且确保调用次数为 1
              regBonusRepo.On("GetStatusAndBonusInCache", mock2.Anything, mock2.Anything, mock2.Anything).Return(
                int16(1), decimal.NewFromInt(12), nil,
              ).Once()
              // 初始化 CaUserClient mock 微服务客户端
              userClient := ca_user_api.CaUserClient(ctx).(*user_proto.MockCasinoUserService)
              // 对 GetExtendInfoList 方法打桩，并返回测试数据，确保调用次数为 1
              userClient.On("GetExtendInfoList", mock2.Anything, mock2.Anything).Return(&user_proto2.GetExtendInfoList_Resp{
                ExtendInfo: map[string]string{
                  causer_enum_proto.USER_EXTEND_KEY_KYC_COMPLETD_REWARD_BONUS.StringAlias(): "12.1",
                },
              }, nil).Once()
              // 对 方法所属结构体的参数进行赋值
              fd := fields{
                registerRepo: regBonusRepo,
              }
              return ctx, fd
            },
          },
          {
            name: "status=1 and bonus=12 and awardStatus=0",
            args: args{
              ctx: trace.GetTraceCtx(metadata.SetCtxData(context.TODO(), metadata.CONTEXT_USER_ID, int64(1234))),
            },
            wantActStatus:        activity_proto.KYC_ACTIVITY_STATUS_ON,
            wantActReward:        "12",
            wantUserRewardStatus: activity_proto.KYC_REWARD_STATUS_UNCOMPLETED,
            wantErr:              false,
            mockInit: func(ctx context.Context, t *testing.T) (context.Context, fields) {
              ctx = mock.MockSetUp(ctx, t)
              // mock registerRepo
              regBonusRepo := dao.NewMockActivityRegisterBonusRepo(t)
              regBonusRepo.On("GetStatusAndBonusInCache", mock2.Anything, mock2.Anything, mock2.Anything).Return(int16(1), decimal.NewFromInt(12), nil).Once()
              // mock CaUserClient
              userClient := ca_user_api.CaUserClient(ctx).(*user_proto.MockCasinoUserService)
              // mock GetExtendInfoList bonus 12.1
              userClient.On("GetExtendInfoList", mock2.Anything, mock2.Anything).Return(&user_proto2.GetExtendInfoList_Resp{
                ExtendInfo: map[string]string{
                  causer_enum_proto.USER_EXTEND_KEY_KYC_COMPLETD_REWARD_BONUS.StringAlias(): "0",
                },
              }, nil).Once()
    
              fd := fields{
                registerRepo: regBonusRepo,
              }
              return ctx, fd
            },
          },
          {
            name: "status=1 and bonus=12 and awardStatus=0 award=null",
            args: args{
              ctx: trace.GetTraceCtx(metadata.SetCtxData(context.TODO(), metadata.CONTEXT_USER_ID, int64(1234))),
            },
            wantActStatus:        activity_proto.KYC_ACTIVITY_STATUS_ON,
            wantActReward:        "12",
            wantUserRewardStatus: activity_proto.KYC_REWARD_STATUS_UNCOMPLETED,
            wantErr:              false,
            mockInit: func(ctx context.Context, t *testing.T) (context.Context, fields) {
              ctx = mock.MockSetUp(ctx, t)
              // mock registerRepo
              regBonusRepo := dao.NewMockActivityRegisterBonusRepo(t)
              regBonusRepo.On("GetStatusAndBonusInCache", mock2.Anything, mock2.Anything, mock2.Anything).Return(int16(1), decimal.NewFromInt(12), nil).Once()
              // mock CaUserClient
              userClient := ca_user_api.CaUserClient(ctx).(*user_proto.MockCasinoUserService)
              // mock GetExtendInfoList bonus 12.1
              userClient.On("GetExtendInfoList", mock2.Anything, mock2.Anything).Return(&user_proto2.GetExtendInfoList_Resp{
                ExtendInfo: map[string]string{
                  causer_enum_proto.USER_EXTEND_KEY_KYC_COMPLETD_REWARD_BONUS.StringAlias(): "null",
                },
              }, nil).Once()
    
              userClient.On("CheckKycPassed", mock2.Anything, mock2.Anything).Return(&user_proto2.CheckKycPassed_Resp{
                Passed: false,
              }, nil).Once()
    
              fd := fields{
                registerRepo: regBonusRepo,
              }
              return ctx, fd
            },
          },
          {
            name: "status=1 and bonus=12 and awardStatus=1 award=null",
            args: args{
              ctx: trace.GetTraceCtx(metadata.SetCtxData(context.TODO(), metadata.CONTEXT_USER_ID, int64(1234))),
            },
            wantActStatus:        activity_proto.KYC_ACTIVITY_STATUS_ON,
            wantActReward:        "12",
            wantUserRewardStatus: activity_proto.KYC_REWARD_STATUS_COMPLETED,
            wantErr:              false,
            mockInit: func(ctx context.Context, t *testing.T) (context.Context, fields) {
              ctx = mock.MockSetUp(ctx, t)
              // mock registerRepo
              regBonusRepo := dao.NewMockActivityRegisterBonusRepo(t)
              regBonusRepo.On("GetStatusAndBonusInCache", mock2.Anything, mock2.Anything, mock2.Anything).Return(
                int16(1), //活动开启
                decimal.NewFromInt(12),
                nil,
              ).Once()
              // mock CaUserClient
              userClient := ca_user_api.CaUserClient(ctx).(*user_proto.MockCasinoUserService)
              // mock GetExtendInfoList bonus 12.1
              userClient.On("GetExtendInfoList", mock2.Anything, mock2.Anything).Return(&user_proto2.GetExtendInfoList_Resp{
                ExtendInfo: map[string]string{
                  causer_enum_proto.USER_EXTEND_KEY_KYC_COMPLETD_REWARD_BONUS.StringAlias(): "null",
                },
              }, nil).Once()
    
              userClient.On("CheckKycPassed", mock2.Anything, mock2.Anything).Return(&user_proto2.CheckKycPassed_Resp{
                Passed: true,
              }, nil).Once()
    
              fd := fields{
                registerRepo: regBonusRepo,
              }
              return ctx, fd
            },
          },
          {
            name: "用户奖励格式不对，返回 error",
            args: args{
              ctx: trace.GetTraceCtx(metadata.SetCtxData(context.TODO(), metadata.CONTEXT_USER_ID, int64(1234))),
            },
            wantActStatus:        activity_proto.KYC_ACTIVITY_STATUS_OFF,
            wantActReward:        "",
            wantUserRewardStatus: activity_proto.KYC_REWARD_STATUS_UNCOMPLETED,
            wantErr:              true,
            mockInit: func(ctx context.Context, t *testing.T) (context.Context, fields) {
              ctx = mock.MockSetUp(ctx, t)
              // mock registerRepo
              regBonusRepo := dao.NewMockActivityRegisterBonusRepo(t)
              regBonusRepo.On("GetStatusAndBonusInCache", mock2.Anything, mock2.Anything, mock2.Anything).Return(
                int16(1), //活动开启
                decimal.NewFromInt(12),
                nil,
              ).Once()
              // mock CaUserClient
              userClient := ca_user_api.CaUserClient(ctx).(*user_proto.MockCasinoUserService)
              // mock GetExtendInfoList bonus 12.1
              userClient.On("GetExtendInfoList", mock2.Anything, mock2.Anything).Return(&user_proto2.GetExtendInfoList_Resp{
                ExtendInfo: map[string]string{
                  causer_enum_proto.USER_EXTEND_KEY_KYC_COMPLETD_REWARD_BONUS.StringAlias(): "t554",
                },
              }, nil).Once()
    
              fd := fields{
                registerRepo: regBonusRepo,
              }
              return ctx, fd
            },
          },
        }
        for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
            //初始化 mock
            tt.args.ctx, tt.fields = tt.mockInit(tt.args.ctx, t)
            a := activityRegister{
              registerRepo: tt.fields.registerRepo,
            }
            gotActStatus, gotActReward, gotUserRewardStatus, err := a.GetKYCCompleteAct(tt.args.ctx)
            if (err != nil) != tt.wantErr {
              t.Errorf("GetKYCCompleteAct() error = %v, wantErr %v", err, tt.wantErr)
              return
            }
            if gotActStatus != tt.wantActStatus {
              t.Errorf("GetKYCCompleteAct() gotActStatus = %v, wantActStatus = %v", gotActStatus, tt.wantActStatus)
            }
            if gotActReward != tt.wantActReward {
              t.Errorf("GetKYCCompleteAct() gotActReward = %v, wantActReward = %v", gotActReward, tt.wantActReward)
            }
            if gotUserRewardStatus != tt.wantUserRewardStatus {
              t.Errorf("GetKYCCompleteAct() gotUserRewardStatus = %v, wantUserRewardStatus = %v", gotUserRewardStatus, tt.wantUserRewardStatus)
            }
    
          })
        }
      }
    ```
