// Package mysql
// @Author: yinwei
// @File: mysql_test.go
// @Version: 1.0.0
// @Date: 2024/9/10 16:25

package mysql

import (
	"context"
	"errors"
	"fmt"
	"github.com/coderyw/gomicro-util/internal/testenv"
	"sync"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/trace"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestMysql(t *testing.T) {
	Init(t)
	ok := testenv.DBConn(t, "OK_GAME_DB")
	c, _, err := InitConnect("test", config.DbConfig{
		Default: "core",
		Instance: map[string]config.DbInstanceConfig{
			"core": {
				DbBaseInstanceConfig: config.DbBaseInstanceConfig{
					Host:        ok.Host,
					Port:        "3261",
					User:        ok.User,
					Password:    ok.Password,
					Database:    "ca_default",
					Driver:      "mysql",
					Open:        50,
					Idle:        20,
					MaxLifeTime: -1,
					//SSLConfig: &config.SSLConfig{
					//	Enable: true,
					//	Path:   "/Users/weienenp/Downloads/ap-southeast-1-bundle.pem",
					//},
				},
				SnowflakeGen: &config.SnowflakeGen{
					//Snowflakes: []config.Snowflake{
					//	{
					//		DataCenter: "b_ho_order",
					//	},
					//},
				},
			},
			"game": {
				DbBaseInstanceConfig: config.DbBaseInstanceConfig{
					Host:        ok.Host,
					Port:        "3261",
					User:        ok.User,
					Password:    ok.Password,
					Database:    "ca_default",
					Driver:      "mysql",
					Open:        50,
					Idle:        20,
					MaxLifeTime: -1,
					//SSLConfig: &config.SSLConfig{
					//	Enable: true,
					//	Path:   "/Users/weienenp/Downloads/ap-southeast-1-bundle.pem",
					//},
				},
				//SnowflakeGen: &config.SnowflakeGen{
				//	Snowflakes: []config.Snowflake{
				//		{
				//			DataCenter: "b_ho_order1",
				//		},
				//	},
				//},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
		return
	}
	cs := cron.New()
	cs.AddFunc(c.CronSpc, c.DoJob)
	go cs.Run()
	time.Sleep(1 * time.Second)
	// 初始化表
	//DB(context.Background()).AutoMigrate(BHoOrder{})
	type args struct {
		serverName string
		conf       config.DbConfig
	}
	ctx := trace.GetTraceCtx(metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok"))
	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{name: "Exec", run: func() error {
			return DB(ctx).Transaction(func(tx *gorm.DB) error {
				time.Sleep(20 * time.Millisecond)
				return tx.Exec("select 1").Error
			})
		}, wantErr: false},
		{name: "SelectFirst", run: func() error {
			obj := BHoOrder{}
			err := DB(ctx).Where("id= ?", 10518784095813637).First(&obj).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("Select", zap.Any("BHoOrder", obj))
			return nil
		}, wantErr: false},
		{name: "SelectByTablename", run: func() error {
			obj := BHoOrder{}
			err := DB(ctx).Table("ca_game_hilo.b_ho_order").Where("id= ?", 10518784095813637).First(&obj).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("Select", zap.Any("BHoOrder", obj))
			return nil
		}, wantErr: false},
		{name: "SelectBatchScanModel", run: func() error {
			var objs []BHoOrder
			err := DB(ctx).Model(BHoOrder{}).Where("sys_user_id= ?", 10501279860129797).Limit(10).Scan(&objs).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("SelectBatch", zap.Any("BHoOrders", objs))
			return nil
		}, wantErr: false},
		{name: "SelectBatchScanTable", run: func() error {
			var objs []BHoOrder
			err := DB(ctx).Table((&BHoOrder{}).TableName()).Where("sys_user_id= ?", 10501279860129797).Limit(10).Scan(&objs).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("SelectBatch", zap.Any("BHoOrders", objs))
			return nil
		}, wantErr: false},
		{name: "SelectBatchFind", run: func() error {
			var objs []BHoOrder
			err := DB(ctx).Where("sys_user_id= ?", 10501279860129797).Limit(10).Find(&objs).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("SelectBatch", zap.Any("BHoOrders", objs))
			return nil
		}, wantErr: false},
		{name: "SelectByChild", run: func() error {
			var objs []BHoOrder
			db := DB(ctx)
			err := db.Where("id in (?)", db.Table("ca_game_hilo.b_ho_order").Select("id").Where("sys_user_id=?", 1)).Find(&objs).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("SelectByChild", zap.Any("BHoOrders", objs))
			return nil
		}, wantErr: false},
		{name: "SelectByRaw", run: func() error {
			var objs = new(BHoOrder)
			err := DB(ctx).Raw("select * from ca_game_hilo.b_ho_order where  id = ?", 198516602105233414).Scan(objs).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("SelectByRaw", zap.Any("BHoOrders", objs))
			return nil
		}, wantErr: false},
		{name: "SelectJoinByModel", run: func() error {
			var objs = new(BHoOrder)
			err := DB(ctx).Joins("JoinTable").Where("b_ho_order.id=1").First(objs).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("SelectJoinByRaw", zap.Any("BHoOrders", objs))
			return nil
		}, wantErr: false},
		{name: "SelectJoinByRaw", run: func() error {
			var objs = new(BHoOrder)
			err := DB(ctx).Raw("select bho.* from ca_game_hilo.b_ho_order bho left join ca_game_hilo.join_table jt on jt.id = bho.id where  bho.id = ?", 198516602105233414).Scan(objs).Error
			if err != nil {
				if errors.Is(err, NotFound) {
					return nil
				}
				return err
			}
			log.Info("SelectJoinByRaw", zap.Any("BHoOrders", objs))
			return nil
		}, wantErr: false},
		{name: "UpdateByStruct", run: func() error {
			obj := BHoOrder{
				BetAmount: 234,
				Status:    1,
			}
			err := DB(ctx).Where("id= ?", 13427739147112453).Updates(obj).Error
			if err != nil {
				return err
			}
			log.Info("UpdateByStruct", zap.Any("BHoOrder", obj))
			return nil
		}, wantErr: false},
		{name: "UpdateByMap", run: func() error {
			obj := map[string]interface{}{
				"bet_amount": 343,
				"status":     3,
			}
			err := DB(ctx).Model(BHoOrder{}).Where("id= ?", 13427739147112453).Updates(obj).Error
			if err != nil {
				return err
			}
			log.Info("UpdateByStruct", zap.Any("BHoOrder", obj))
			return nil
		}, wantErr: false},
		{name: "insertByStruct", run: func() error {
			obj := &BHoOrder{
				BetAmount: 234,
				Status:    1,
			}
			err := DB(ctx, "game").Create(obj).Error
			if err != nil {
				return err
			}
			log.Info("insertByStruct", zap.Any("BHoOrder", obj))
			return nil
		}, wantErr: false},
		//{name: "insertBySql", run: func() error {
		//	err := DB(ctx).
		//		Exec("INSERT INTO ca_game_hilo.b_ho_order (`trace_id`,`sys_user_id`,`status`,`currency`,`game_id`,`bet_amount`,`transfer_amount`,`req_id`,`third_create_time`,`third_updated_time`,`created_at`,`updated_at`,`start_balance`,`after_balance`,`end_balance`,`extra_1`,`extra_2`) VALUES ('',0,1,'',0,234,0,'',0,0,'2026-03-12 13:48:29.203','0000-00-00 00:00:00',0,0,0,'','')").Error
		//
		//	if err != nil {
		//		return err
		//	}
		//	return nil
		//}, wantErr: false},
		{name: "insertBatchByStruct", run: func() error {
			obj := []*BHoOrder{&BHoOrder{
				BetAmount: 234,
				Status:    1,
			}, &BHoOrder{
				BetAmount: 445,
				Status:    3,
			}}
			err := DB(ctx).Create(obj).Error
			if err != nil {
				return err
			}
			log.Info("insertByStruct", zap.Any("BHoOrder", obj))
			return nil
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.wantErr != (err != nil) {
				t.Error(err.Error())
			}
		})
	}
}

type DwsPlayerGameDayStats struct {
	Date      string `json:"date"`
	UserId    int64  `json:"user_id"`
	GameId    int64  `json:"game_id"`
	BetAmount int64  `json:"bet_amount"`
}

func (d *DwsPlayerGameDayStats) TableName() string {
	return "dws.dws_player_game_day_stats"
}

func TestGaussDb(t *testing.T) {
	Init(t)
	gauss := testenv.DBConn(t, "GAUSS_TEST_DB")
	c, _, err := InitConnect("test", config.DbConfig{
		Default: "core",
		Instance: map[string]config.DbInstanceConfig{
			"core": {
				DbBaseInstanceConfig: config.DbBaseInstanceConfig{
					Host:        gauss.Host,
					Port:        "8000",
					User:        gauss.User,
					Password:    gauss.Password,
					Database:    "ns_db",
					Driver:      "opengauss",
					Open:        50,
					Idle:        20,
					MaxLifeTime: -1,
				},
				SnowflakeGen: &config.SnowflakeGen{
					Snowflakes: []config.Snowflake{
						{
							DataCenter: "",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
		return
	}
	cs := cron.New()
	cs.AddFunc(c.CronSpc, c.DoJob)
	go cs.Run()
	time.Sleep(1 * time.Second)
	type args struct {
		serverName string
		conf       config.DbConfig
	}
	ctx := trace.GetTraceCtx(context.WithValue(context.Background(), "Custom_tenant", "test"))
	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{name: "Exec", run: func() error {
			return DB(ctx).Transaction(func(tx *gorm.DB) error {
				time.Sleep(20 * time.Millisecond)
				return tx.Exec("select 1").Error
			})
		}, wantErr: false},
		{name: "SelectFirst", run: func() error {
			obj := DwsPlayerGameDayStats{}
			err := DB(ctx).Where("date= ? and user_id=?", "2025-10-27", 56125261191663626).First(&obj).Error
			if err != nil {
				return err
			}
			log.Info("Select", zap.Any("DwsPlayerGameDayStats", obj))
			return nil
		}, wantErr: false},
		{name: "SelectBatch", run: func() error {
			var objs []DwsPlayerGameDayStats
			err := DB(ctx).Where("date= ?", "2025-10-27").Limit(10).Find(&objs).Error
			if err != nil {
				return err
			}
			log.Info("SelectBatch", zap.Any("DwsPlayerGameDayStats", objs))
			return nil
		}, wantErr: false},
		{name: "UpdateByStruct", run: func() error {
			obj := DwsPlayerGameDayStats{
				BetAmount: 234,
			}
			err := DB(ctx).Where("date= ? and user_id=?", "2025-10-27", 56125261191663626).Updates(obj).Error
			if err != nil {
				return err
			}
			log.Info("UpdateByStruct", zap.Any("DwsPlayerGameDayStats", obj))
			return nil
		}, wantErr: false},
		{name: "UpdateByMap", run: func() error {
			obj := map[string]interface{}{
				"bet_amount": 22222,
			}
			err := DB(ctx).Model(DwsPlayerGameDayStats{}).Where("date= ? and user_id=?", "2025-10-27", 56125261191663626).Updates(obj).Error
			if err != nil {
				return err
			}
			log.Info("UpdateByStruct", zap.Any("DwsPlayerGameDayStats", obj))
			return nil
		}, wantErr: false},
		{name: "insertByStruct", run: func() error {
			obj := &DwsPlayerGameDayStats{
				BetAmount: 234,
				Date:      "2025-12-31",
				GameId:    12,
				UserId:    56125261191663626,
			}
			err := DB(ctx).Create(obj).Error
			if err != nil {
				return err
			}
			log.Info("insertByStruct", zap.Any("DwsPlayerGameDayStats", obj))
			return nil
		}, wantErr: false},
		{name: "insertBatchByStruct", run: func() error {
			obj := []*DwsPlayerGameDayStats{&DwsPlayerGameDayStats{
				BetAmount: 232,
				Date:      "2025-12-30",
				GameId:    12,
				UserId:    56125261191663626,
			}, &DwsPlayerGameDayStats{
				BetAmount: 333,
				Date:      "2025-12-30",
				GameId:    12,
				UserId:    56125261191663621,
			}}
			err := DB(ctx).Create(obj).Error
			if err != nil {
				return err
			}
			log.Info("insertByStruct", zap.Any("DwsPlayerGameDayStats", obj))
			return nil
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.wantErr != (err != nil) {
				t.Error(err.Error())
			}
		})
	}
}

func TestBatch(t *testing.T) {
	caUser := testenv.DBConn(t, "CA_USER_DB")
	type args struct {
		serverName string
		conf       config.DbConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "测试链接gaussdb", args: args{
			serverName: "测试",
			conf: config.DbConfig{
				Default: "core",
				Instance: map[string]config.DbInstanceConfig{
					"core": {
						DbBaseInstanceConfig: config.DbBaseInstanceConfig{
							Host:        caUser.Host,
							Port:        "5066",
							User:        caUser.User,
							Password:    caUser.Password,
							Database:    "ca_user",
							Driver:      "mysql",
							Open:        50,
							Idle:        20,
							MaxLifeTime: -1,
						},
					},
				},
			},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := InitConnect(tt.args.serverName, tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitConnect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			time.Sleep(1 * time.Second)
			wg := sync.WaitGroup{}
			for i := 0; i < 10000; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err = DB(trace.GetTraceCtx(context.Background())).Transaction(func(tx *gorm.DB) error {
						time.Sleep(20 * time.Millisecond)
						return tx.Exec("select 1").Error
					})
				}()
			}
			wg.Wait()

			//err = DB(ctx).Transaction(func(tx *gorm.DB) error {
			//	return DB(ctx).Exec("select 1").Error
			//	return nil
			//})
			if err != nil {
				fmt.Println(err)
			}
		})
	}
}

func TestBatchQuery(t *testing.T) {
	caUser := testenv.DBConn(t, "CA_USER_DB")
	type args struct {
		serverName string
		conf       config.DbConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "测试链接gaussdb", args: args{
			serverName: "测试",
			conf: config.DbConfig{
				Default: "core",
				Instance: map[string]config.DbInstanceConfig{
					"core": {
						DbBaseInstanceConfig: config.DbBaseInstanceConfig{
							Host:     caUser.Host,
							Port:     "5066",
							User:     caUser.User,
							Password: caUser.Password,
							Database: "ca_user",
							Driver:   "mysql",
						},
					},
				},
			},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := InitConnect(tt.args.serverName, tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitConnect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			wg := &sync.WaitGroup{}
			start := time.Now()
			for _ = range 200 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					DB(trace.GetTraceCtx(context.Background())).Exec("SELECT SLEEP(0.02);")
				}()
			}
			wg.Wait()
			log.Info("执行完耗时", zap.Any("耗时", time.Since(start)))
		})
	}
}

func TestMulSource(t *testing.T) {
	release := testenv.DBConn(t, "RELEASE_GAUSS_DB")
	type args struct {
		serverName string
		conf       config.DbConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "测试链接gaussdb", args: args{
			serverName: "测试",
			conf: config.DbConfig{
				Default: "core",
				Instance: map[string]config.DbInstanceConfig{
					"core": {
						DbBaseInstanceConfig: config.DbBaseInstanceConfig{
							Host:     release.Host,
							Port:     "8000",
							User:     release.User,
							Password: release.Password,
							Database: "ca_core",
							Driver:   "opengauss",
						},
					},
					"ca_game_base": {
						DbBaseInstanceConfig: config.DbBaseInstanceConfig{
							Host:     release.Host,
							Port:     "8000",
							User:     release.User,
							Password: release.Password,
							Database: "ca_game",
							Driver:   "opengauss",
						},
					},
				},
			},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := InitConnect(tt.args.serverName, tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitConnect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			rows, err := DB(context.Background()).Raw("select id from b_channel where `id` = 1").Rows()
			if err != nil {
				t.Error(err.Error())
				return
			}
			for rows.Next() {
				var id int64
				rows.Scan(&id)
				fmt.Println(id)
				break
			}

			rows, err = DB(context.Background(), "ca_game_base").Raw("select id from g_game_svrs where id = 98").Rows()
			if err != nil {
				t.Error(err.Error())
				return
			}
			for rows.Next() {
				var id int64
				rows.Scan(&id)
				fmt.Println(id)
				break
			}
		})
	}
}

type BBalance struct {
	ID        uint  `json:"id" gorm:"column:id"`
	PlayerID  uint  `json:"player_id" gorm:"column:player_id"`   // 用户id
	Type      int   `json:"type" gorm:"column:type"`             // 余额类型1可提现2不可提现(暂时没有这个限制，都是2)
	Balance   int64 `json:"balance" gorm:"column:balance"`       // 余额
	ClubID    int   `json:"club_id" gorm:"column:club_id"`       // 俱乐部ID
	UpdatedAt uint  `json:"updated_at" gorm:"column:updated_at"` // 更新时间
}

func (m *BBalance) TableName() string {
	return "ca_balance.b_balance"
}

func GetBalance(db *gorm.DB) *BBalance {
	var info = new(BBalance)
	err := db.Model(info).Where("id=?", 1).First(&info).Error
	if err != nil {
		return nil
	}
	return info
}
