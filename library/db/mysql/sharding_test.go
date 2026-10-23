// Package mysql
// @Author: yinwei
// @File: sharding_test
// @Version: 1.0.0
// @Date: 2024/9/24 09:19

package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/xwb1989/sqlparser"
	"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"github.com/coderyw/gomicro-util/model/config"
	"gorm.io/gorm"
)

type BHoOrder struct {
	ID               int64     `json:"id" gorm:"column:id"`
	TraceID          string    `json:"trace_id" gorm:"column:trace_id"`
	SysUserID        uint64    `json:"sys_user_id" gorm:"column:sys_user_id"`                    // 系统user_id
	Status           uint8     `json:"status" gorm:"column:status"`                              // 注单状态 0：未成功，1:注单成功，2：取消注单
	Currency         string    `json:"currency" gorm:"column:currency"`                          // 货币名称
	GameID           int64     `json:"game_id" gorm:"column:game_id"`                            // 游戏代码 对方游戏id
	BetAmount        int64     `json:"bet_amount" gorm:"column:bet_amount"`                      // 押注金额
	TransferAmount   int64     `json:"transfer_amount" gorm:"column:transfer_amount"`            // 赢得金额
	ReqId            string    `json:"req_id" gorm:"column:req_id"`                              // 交易的唯一标识符
	ThirdCreateTime  int64     `json:"third_create_time" gorm:"column:third_create_time"`        // 毫秒 对方的创建时间
	ThirdUpdatedTime int64     `json:"third_updated_time" gorm:"column:third_updated_time"`      // 毫秒 不等于投注结束时间
	CreatedAt        time.Time `json:"created_at" gorm:"column:created_at"`                      // 创建时间
	UpdatedAt        time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime:false"` // 更新时间
	StartBalance     int64     `json:"start_balance" gorm:"start_balance"`                       // 下注前金额
	AfterBalance     int64     `json:"after_balance" gorm:"after_balance"`                       // 下注后金额
	EndBalance       int64     `json:"end_balance" gorm:"end_balance"`                           // 结算后金额
	Extra1           string    `json:"extra_1" gorm:"column:extra_1"`
	Extra2           string    `json:"extra_2" gorm:"column:extra_2"`
	JoinTable        JoinTable `gorm:"foreignKey:sys_user_id" structs:"-"`
}

type JoinTable struct {
	ID   int64  `json:"id" gorm:"column:id"`
	Name string `json:"name" gorm:"column:name"`
}

func (u *JoinTable) TableName() string {
	return "ca_game_hilo.join_table"
}

func (m *BHoOrder) TableName() string {
	return "ca_game_hilo.b_ho_order"
	// return "b_ho_order"
}

type BJiliOrders struct {
	ID            int64           `json:"id" gorm:"column:id;"`
	ReqID         string          `json:"req_id" gorm:"column:req_id"`                 // 用户id
	SysUserID     int64           `json:"sys_user_id" gorm:"column:sys_user_id"`       // 系统user_id
	Hall          uint8           `json:"hall" gorm:"column:hall"`                     // 大厅
	Status        uint8           `json:"status" gorm:"column:status"`                 // 注单状态 0：未成功，1:注单成功，2：取消注单
	Currency      string          `json:"currency" gorm:"column:currency"`             // 货币名称
	Game          int             `json:"game" gorm:"column:game"`                     // 游戏代码
	Round         string          `json:"round" gorm:"column:round"`                   // 注单唯一值 (等同 WagersId)
	WagersTime    int64           `json:"wagers_time" gorm:"column:wagers_time"`       // 更新时间
	BetAmount     decimal.Decimal `json:"bet_amount" gorm:"column:bet_amount"`         // 押注金额
	WinloseAmount decimal.Decimal `json:"winlose_amount" gorm:"column:winlose_amount"` // 派彩金额
	IsFreeRound   int8            `json:"is_free_round" gorm:"column:is_free_round"`   //
	UserID        string          `json:"user_id" gorm:"column:user_id"`               // 玩家账号唯一值  依营运商需求带入 离线开奖一定 会带
	TransactionID string          `json:"transaction_id" gorm:"column:transaction_id"` //
	Platform      string          `json:"platform" gorm:"column:platform"`             //
	StatementType int             `json:"statement_type" gorm:"column:statement_type"` //
	SummaryID     uint            `json:"summary_id" gorm:"column:summary_id"`         // 汇总表id
	GameCategory  int             `json:"game_category" gorm:"column:game_category"`   //
	CancelInfo    string          `json:"cancel_info" gorm:"column:cancel_info"`       //
	CreatedAt     time.Time       `json:"created_at" gorm:"column:created_at"`         // 创建时间
	UpdatedAt     time.Time       `json:"updated_at" gorm:"column:updated_at"`         // 更新时间
	StartBalance  decimal.Decimal `json:"start_balance" gorm:"column:start_balance"`
	AfterBalance  decimal.Decimal `json:"after_balance" gorm:"column:after_balance"`
	EndBalance    decimal.Decimal `json:"end_balance" gorm:"column:end_balance"`
}

func (m *BJiliOrders) TableName() string {
	return "ca_game_jili.b_jili_orders"
}

func Init() {
	_, _, err := redis.InitRedis(config.RedisConfig{
		Enable:  true,
		Default: "core",
		Instance: map[string]config.RedisInstanceConfig{
			"core": {
				Enable:    true,
				Host:      []string{"clustercfg.dev-valkey-cfn.dkgdhb.apse1.cache.amazonaws.com:6379"},
				Password:  "Jr6s7UvKcsbhkA4UF6",
				Db:        0,
				IsCluster: true,
			},
			"snowflake": {
				Enable:    true,
				Host:      []string{"clustercfg.dev-valkey-cfn.dkgdhb.apse1.cache.amazonaws.com:6379"},
				Password:  "Jr6s7UvKcsbhkA4UF6",
				Db:        0,
				IsCluster: true,
			},
		},
	})
	if err != nil {
		panic(err)
	}
}

func TestGaussSharding(t *testing.T) {
	Init()

	conf := config.DbConfig{
		Default: "core",
		Instance: map[string]config.DbInstanceConfig{
			"core": {
				DbBaseInstanceConfig: config.DbBaseInstanceConfig{
					Host:     "192.168.64.62",
					Port:     "5066",
					User:     "core_dev_reader",
					Password: "keWx6LSGku0VAY6t#YL",
					Database: "ca_user",
					Driver:   "mysql",
					Open:     100,
					Idle:     20,
					Debug:    true,
				},
				Sharding: []config.ShardingTable{
					{TableName: "ca_game_hilo.b_ho_order", ShardingKey: "sys_user_id", ShardingNum: 100},
					{TableName: "b_ho_order", ShardingKey: "sys_user_id", ShardingNum: 100},
				},
				SnowflakeGen: &config.SnowflakeGen{
					Snowflakes: []config.Snowflake{
						{
							DataCenter: "test-1",
						},
					},
				},
			},
		},
	}
	InitConnect("test", conf)

	t.Run("创建", func(t *testing.T) {
		val := &BHoOrder{
			ID:               213,
			TraceID:          "fwigi3j2903rh2i",
			SysUserID:        12,
			Status:           1,
			Currency:         "PHP",
			GameID:           1001,
			BetAmount:        1200,
			TransferAmount:   1200,
			ReqId:            "",
			ThirdCreateTime:  0,
			ThirdUpdatedTime: 0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			StartBalance:     1,
			AfterBalance:     2,
			EndBalance:       3,
			Extra1:           "",
			Extra2:           "",
		}
		err := DB(context.Background()).Transaction(func(tx *gorm.DB) error {
			// sql := tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Create(val).Error
			//})
			//t.Log(sql)
			//return tx.Exec(sql).Error
		})
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("批量创建[]BHoOrder", func(t *testing.T) {
		val := BHoOrder{
			TraceID:          "fwigi3j2903rh2i",
			SysUserID:        12,
			Status:           1,
			Currency:         "PHP",
			GameID:           1001,
			BetAmount:        1200,
			TransferAmount:   1200,
			ReqId:            "",
			ThirdCreateTime:  0,
			ThirdUpdatedTime: 0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			StartBalance:     1,
			AfterBalance:     2,
			EndBalance:       3,
			Extra1:           "",
			Extra2:           "",
		}
		var arr []BHoOrder
		for i := 0; i < 10; i++ {
			t := val
			t.TraceID = fmt.Sprintf("%d", i)
			t.BetAmount = int64(i)
			arr = append(arr, t)
		}

		err := DB(context.Background()).Create(&arr).Error
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("批量创建[]*BHoOrder", func(t *testing.T) {
		val := BHoOrder{
			TraceID:          "fwigi3j2903rh2i",
			SysUserID:        12,
			Status:           1,
			Currency:         "PHP",
			GameID:           1001,
			BetAmount:        1200,
			TransferAmount:   1200,
			ReqId:            "",
			ThirdCreateTime:  0,
			ThirdUpdatedTime: 0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			StartBalance:     1,
			AfterBalance:     2,
			EndBalance:       3,
			Extra1:           "",
			Extra2:           "",
		}
		var arr []*BHoOrder
		for i := 0; i < 10; i++ {
			t := val
			t.SysUserID = uint64(i)
			t.TraceID = fmt.Sprintf("%d", i)
			t.BetAmount = int64(i)
			arr = append(arr, &t)
		}

		err := DB(context.Background()).Create(&arr).Error
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("创建Map", func(t *testing.T) {
		val := map[string]interface{}{
			"trace_id":    "213jiwfjwefi",
			"sys_user_id": 12,
			"bet_amount":  12,
			"req_id":      "werwe",
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		}
		err := DB(context.Background()).Model(BHoOrder{}).Create(val).Error
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("更新", func(t *testing.T) {
		val := &BHoOrder{
			BetAmount: 333333,
		}
		err := DB(context.Background()).Where("id=? and sys_user_id=?", int64(24979912838348805), 12).Updates(val).Error
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("查询第一个", func(t *testing.T) {
		val := new(BHoOrder)
		err := DB(context.Background()).Where("sys_user_id=?", 12).First(val).Error
		if err != nil {
			t.Error(err)
		} else {
			t.Log(val)
		}
	})

	t.Run("join", func(t *testing.T) {
		DB(context.Background()).Where("sys_user_id=?", 12).Joins("")
	})
}

func TestParse(t *testing.T) {
	r := "INSERT INTO ag.table1 VALUES (1, 'a'); "
	expr, err := sqlparser.Parse(r)
	if err != nil {
		t.Fatal(err)
	}
	switch stmt := expr.(type) {
	case *sqlparser.Select:

	case *sqlparser.Insert:
		fmt.Println(stmt.Table.Name)
		fmt.Println(stmt)
	}
}
