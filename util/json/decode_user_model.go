// Package mapstructure
// @Author: yinwei
// @File: decode_test_model
// @Version: 1.0.0
// @Date: 2025/4/23 13:07

package json

import "github.com/shopspring/decimal"

type UserCacheGo struct {
	ID                  *int64          `gorm:"column:id;type:int unsigned;primaryKey;autoIncrement:true" json:"id"`
	DeviceID            string          `gorm:"column:device_id;type:varchar(64);not null;comment:设备id" json:"device_id"`                       // 设备id
	AppChannel          string          `gorm:"column:app_channel;type:varchar(64);not null;comment:用户注册渠道" json:"app_channel"`                 // 用户注册渠道
	RegistrationChannel string          `gorm:"column:registration_channel;type:varchar(64);not null;comment:注册渠道" json:"registration_channel"` // 注册渠道
	LastLoginChannel    string          `gorm:"column:last_login_channel;type:varchar(64);not null;comment:最后登录渠道" json:"last_login_channel"`   // 最后登录渠道
	AppPackageName      string          `gorm:"column:app_package_name;type:varchar(64);not null;comment:包名" json:"app_package_name"`           // 包名
	AppVersion          string          `gorm:"column:app_version;type:varchar(64);not null;comment:版本" json:"app_version"`                     // 版本
	Source              int64           `gorm:"column:source;type:int unsigned;not null" json:"source"`
	Phone               string          `gorm:"column:phone;type:varchar(64);not null;comment:手机号" json:"phone"`                                         // 手机号
	Password            string          `gorm:"column:password;type:varchar(64);not null;comment:密码" json:"password"`                                    // 密码
	WithdrawPassword    string          `gorm:"column:withdraw_password;type:varchar(64);comment:提现密码" json:"withdraw_password"`                         // 提现密码
	Salt                string          `gorm:"column:salt;type:varchar(64);not null;comment:密码盐值" json:"salt"`                                          // 密码盐值
	FbToken             string          `gorm:"column:fb_token;type:varchar(64);not null;comment:face-token" json:"fb_token"`                            // face-token
	GoogleToken         string          `gorm:"column:google_token;type:varchar(64);not null;comment:google-token" json:"google_token"`                  // google-token
	GcashCustomerID     string          `gorm:"column:gcash_customer_id;type:varchar(64);not null;comment:gcash-customer-id唯一" json:"gcash_customer_id"` // gcash-customer-id唯一
	MayaCustomerID      string          `gorm:"column:maya_customer_id;type:varchar(64);not null;comment:maya小程序唯一ID" json:"maya_customer_id"`           // maya小程序唯一ID
	MayaPhone           string          `gorm:"column:maya_phone;type:varchar(64);not null;comment:maya小程序手机号" json:"maya_phone"`                        // maya小程序手机号
	GcashPhone          string          `gorm:"column:gcash_phone;type:varchar(64);not null;comment:gcash小程序手机号" json:"gcash_phone"`                     // gcash小程序手机号
	Nickname            string          `gorm:"column:nickname;type:varchar(64);not null;comment:昵称" json:"nickname"`                                    // 昵称
	Avatar              string          `gorm:"column:avatar;type:varchar(256);not null;comment:头像" json:"avatar"`                                       // 头像
	FbAvatar            string          `gorm:"column:fb_avatar;type:varchar(256);not null;comment:facebook头像" json:"fb_avatar"`                         // facebook头像
	AvatarFrame         int64           `gorm:"column:avatar_frame;type:int;not null" json:"avatar_frame"`
	CardBack            int64           `gorm:"column:card_back;type:int;not null" json:"card_back"`
	PayAccountID        int64           `gorm:"column:pay_account_id;type:int unsigned;not null;comment:用户支付账号id" json:"pay_account_id"`                          // 用户支付账号id
	Status              int8            `gorm:"column:status;type:tinyint;not null;comment:用户状态0默认正常 -1 禁止登录 2 管理员封禁 3 用户自己冻结 4后台删除（后台不显示，测试用户特有）" json:"status"` // 用户状态0默认正常 -1 禁止登录 2 管理员封禁 3 用户自己冻结 4后台删除（后台不显示，测试用户特有）
	FirebaseToken       string          `gorm:"column:firebase_token;type:varchar(256);not null;comment:firabase_token" json:"firebase_token"`                    // firabase_token
	InviteUserID        int64           `gorm:"column:invite_user_id;type:int unsigned;not null;comment:邀请人用户id" json:"invite_user_id"`                           // 邀请人用户id
	InviteAt            int64           `gorm:"column:invite_at;type:int unsigned;not null" json:"invite_at"`
	InviteLevel         int64           `gorm:"column:invite_level;type:int unsigned;not null" json:"invite_level"`
	CheckinCount        int64           `gorm:"column:checkin_count;type:int unsigned;not null" json:"checkin_count"`
	Tag                 uint8           `gorm:"column:tag;type:tinyint unsigned;not null;comment:用户标签" json:"tag"`                                             // 用户标签
	Language            string          `gorm:"column:language;type:varchar(16);not null;comment:用户设备语言" json:"language"`                                      // 用户设备语言
	IsBusinessman       int64           `gorm:"column:is_businessman;type:int unsigned;not null;comment:是否是商人" json:"is_businessman"`                          // 是否是商人
	WithdrawModel       int64           `gorm:"column:withdraw_model;type:int unsigned;not null;default:1;comment:提现模式，1银商模式，2普通模式" json:"withdraw_model"`     // 提现模式，1银商模式，2普通模式
	Type                string          `gorm:"column:type;type:varchar(64);not null;comment:用户类型" json:"type"`                                                // 用户类型
	RechargeAmount      decimal.Decimal `gorm:"column:recharge_amount;type:decimal(11,2) unsigned;not null;default:0.00;comment:充值总金额" json:"recharge_amount"` // 充值总金额
	CreatedAt           int64           `gorm:"column:created_at;type:int unsigned;not null;comment:创建时间" json:"created_at"`                                   // 创建时间
	UpdatedAt           int64           `gorm:"column:updated_at;type:int unsigned;not null;comment:更新时间;autoUpdateTime:false" json:"updated_at"`              // 更新时间
	LastLoginAt         int64           `gorm:"column:last_login_at;type:int;not null;comment:最后登录时间" json:"last_login_at"`                                    // 最后登录时间
	RegisterIP          string          `gorm:"column:register_ip;type:varchar(64);not null;comment:注册ip" json:"register_ip"`                                  // 注册ip
	LastLoginIP         string          `gorm:"column:last_login_ip;type:varchar(64);not null;comment:最后登录 IP" json:"last_login_ip"`                           // 最后登录 IP
	WithdrawAmount      decimal.Decimal `gorm:"column:withdraw_amount;type:decimal(11,2) unsigned;not null;default:0.00" json:"withdraw_amount"`
	VipScore            int64           `gorm:"column:vip_score;type:int unsigned;not null;comment:vip" json:"vip_score"`         // vip
	VipLevelMax         int64           `gorm:"column:vip_level_max;type:int unsigned;not null;comment:vip" json:"vip_level_max"` // vip
	StatusRemark        string          `gorm:"column:status_remark;type:varchar(255);comment:状态备注" json:"status_remark"`         // 状态备注
	VipStart            int64           `gorm:"column:vip_start;default:0;NOT NULL" json:"vip_start"`

	VipEnd int64 `gorm:"column:vip_end;default:0;NOT NULL" json:"vip_end"`

	VipLevel int16 `json:"vip_level"`

	Token string `json:"token"`

	UserID        int64   `json:"user_id"`
	Balance       int64   `json:"balance"`
	LoginPassword string  `json:"login_password"`
	Bind          []int64 `json:"bind"`
	InviteCode    string  `json:"invite_code"`
	// todo 当前去掉这个 map 解析
	Invite          *UserTinyInfo   `json:"invite" mapstructure:"-"`
	IsRegister      uint8           `json:"is_register"`
	Enable          uint8           `json:"enable"`
	AvailableCoins  decimal.Decimal `json:"available_coins"`
	SPlayer         int64           `json:"s_player"`
	CPlayer         int64           `json:"c_player"`
	WithdrawControl int             `json:"withdraw_control"`
	TotalRounds     int             `json:"total_rounds"`
	FirstRwReward   bool            `json:"first_rw_reward"`
	Level           int             `json:"level"`
	LevelMax        int             `json:"level_max"`
	Score           int             `json:"score"`
	PayAccount      *PayAccount     `json:"pay_account"`
	Connection      *Connection     `json:"connection"`
	PlayerID        int64           `json:"player_id"`

	Grayscale int64 `json:"grayscale"`
	IsVip     int8  `json:"is_vip"`
}
type PayAccount struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Name  string `json:"name"`
}
type Connection struct {
}

type UserTinyInfo struct {
	UserId     int64  `json:"user_id"`
	Nickname   string `json:"nickname"`
	Avatar     string `json:"avatar"`
	FbAvatar   string `json:"fb_avatar"`
	InviteCode string `json:"invite_code"`
	Phone      string `json:"phone"`
}
