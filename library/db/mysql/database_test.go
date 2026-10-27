package mysql

import (
	"context"
	"github.com/coderyw/gomicro-util/internal/testenv"
	"regexp"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/model/config"
	"gorm.io/gorm"
)

type ctxKey string

const namespaceKey ctxKey = "ns"

func withNS(ctx context.Context, ns string) context.Context {
	return metadata.SetCtxData(ctx, metadata.CONTEXT_NAME_SPACE, ns)
}

type User struct {
	ID       int64    `gorm:"column:id;primaryKey"`
	Name     string   `gorm:"column:name"`
	BVipUser BVipUser `gorm:"foreignKey:UserID;references:ID"`
}

func (User) TableName() string {
	return "ca_game_hilo.b_user"
}

type BVipUser struct {
	ID     int64 `gorm:"column:id;primaryKey"`
	UserID int64 `gorm:"column:user_id"`
	Level  int   `gorm:"column:level"`
}

func (BVipUser) TableName() string {
	return "ca_game_hilo.b_vip_user"
}

func newTestDB(t *testing.T) *gorm.DB {
	Init(t)
	ok := testenv.DBConn(t, "OK_GAME_DB")
	_, _, err := InitConnect("test", config.DbConfig{
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
	}
	return DB(metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok")).Session(&gorm.Session{DryRun: true})
}

func normalizeSQL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

func assertContainsAll(t *testing.T, sql string, parts ...string) {
	t.Helper()
	sql = normalizeSQL(sql)
	for _, p := range parts {
		if !strings.Contains(sql, p) {
			t.Fatalf("sql not contains expected part\nexpected: %q\nactual:   %q", p, sql)
		}
	}
}

var regs = []*regexp.Regexp{
	regexp.MustCompile("`ca_game_hilo`\\s*\\."),
	regexp.MustCompile(`"ca_game_hilo"\s*\.`),
	regexp.MustCompile(`(^|[^a-zA-Z0-9_])ca_game_hilo\s*\.`),
}

// assertOnlyNewDB 校验：
// 1. SQL 中必须出现目标库名 ok_ca_game_hilo
// 2. SQL 中不能再出现“独立的旧库名引用” ca_game_hilo.xxx
func assertOnlyNewDB(t *testing.T, sql string, mustHas string, mustNotHas []*regexp.Regexp) {
	t.Helper()

	sql = normalizeSQL(sql)

	// 必须包含新库名
	if !strings.Contains(sql, mustHas) {
		t.Fatalf("sql does not contain rewritten db name.\nsql=%s", sql)
	}

	// 不能出现旧库名的独立引用：
	// - `ca_game_hilo`.
	// - "ca_game_hilo".
	// - ca_game_hilo.
	//
	// 注意：不能简单 strings.Contains("ca_game_hilo")
	// 因为 ok_ca_game_hilo 本身会包含这个子串
	oldDBPatterns := mustNotHas

	for _, re := range oldDBPatterns {
		if re.FindStringIndex(sql) != nil {
			t.Fatalf("sql still contains old db reference matched by %q.\nsql=%s", re.String(), sql)
		}
	}

	// 额外防重复前缀
	if strings.Contains(sql, "ok_ok_ca_game_hilo") {
		t.Fatalf("sql contains duplicated rewritten db name.\nsql=%s", sql)
	}
}

func TestPlugin_NormalQuery(t *testing.T) {
	db := newTestDB(t)
	ctx := withNS(context.Background(), "ok")

	stmt := db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", 123).
		First(&User{}).Statement

	sql := stmt.SQL.String()

	assertOnlyNewDB(t, sql, "ok_ca_game_hilo", regs)
}

func TestPlugin_SubQuery(t *testing.T) {
	db := newTestDB(t)
	ctx := withNS(context.Background(), "ok")

	sub := db.WithContext(ctx).
		Table("ca_game_hilo.b_ho_order").
		Select("id").
		Where("sys_user_id = ?", 1)

	stmt := db.WithContext(ctx).
		Model(&BHoOrder{}).
		Where("id IN (?)", sub).
		Find(&[]BHoOrder{}).Statement

	sql := stmt.SQL.String()

	assertOnlyNewDB(t, sql, "ok_ca_game_hilo", regs)
}

func TestPlugin_Create(t *testing.T) {
	db := newTestDB(t)
	ctx := withNS(context.Background(), "ok")

	u := User{
		ID:   1,
		Name: "tom",
	}

	stmt := db.WithContext(ctx).Create(&u).Statement
	sql := stmt.SQL.String()

	assertOnlyNewDB(t, sql, "ok_ca_game_hilo", regs)
}

func TestPlugin_Update(t *testing.T) {
	db := newTestDB(t)
	ctx := withNS(context.Background(), "ok")

	stmt := db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", 1).
		Update("name", "jack").Statement

	sql := stmt.SQL.String()

	assertOnlyNewDB(t, sql, "ok_ca_game_hilo", regs)
}

func TestPlugin_Delete(t *testing.T) {
	db := newTestDB(t)
	ctx := withNS(context.Background(), "ok")

	stmt := db.WithContext(ctx).
		Where("id = ?", 1).
		Delete(&User{}).Statement

	sql := stmt.SQL.String()

	assertContainsAll(t, sql,
		"DELETE FROM `ok_ca_game_hilo`.`b_user`",
		"WHERE id = ?",
	)
}

func TestPlugin_Exec(t *testing.T) {
	db := newTestDB(t)
	ctx := withNS(context.Background(), "ok")

	stmt := db.WithContext(ctx).
		Exec("UPDATE ca_game_hilo.b_user SET name=? WHERE id=?", "abc", 1).
		Statement

	sql := stmt.SQL.String()

	assertOnlyNewDB(t, sql, "ok_ca_game_hilo", regs)
}

func TestPlugin_AssociationJoin(t *testing.T) {
	db := newTestDB(t)
	ctx := withNS(context.Background(), "ok")

	var user User
	stmt := db.WithContext(ctx).
		Model(&User{}).
		Joins("BVipUser").
		First(&user, "b_user.id = ?", 100).Statement

	sql := stmt.SQL.String()

	assertOnlyNewDB(t, sql, "ok_ca_game_hilo", regs)
}
