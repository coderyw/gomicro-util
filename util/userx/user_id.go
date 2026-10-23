package userx

import (
	env2 "github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/model/cst"
	"strings"
)

// Username2UserID 第三方UserID转换为我方UserID
func Username2UserID(username string) string {
	// 判断环境
	env := env2.GetEnv()
	if env == cst.RUNTIME_ENV_PRE || env == cst.RUNTIME_ENV_TEST || env == cst.RUNTIME_ENV_DEV || env == cst.RUNTIME_ENV_ROADSHOW {
		suffix := getSuffix()
		if strings.HasSuffix(username, suffix) {
			return username[:len(username)-len(suffix)]
		}
		if strings.HasSuffix(username, strings.ToLower(suffix)) {
			return username[:len(username)-len(strings.ToLower(suffix))]
		}
	}
	return username
}

// UserID2Username 我方UserID转换为第三方Username
func UserID2Username(userID string) string {
	env := env2.GetEnv()
	if env == cst.RUNTIME_ENV_PRE || env == cst.RUNTIME_ENV_TEST || env == cst.RUNTIME_ENV_DEV || env == cst.RUNTIME_ENV_ROADSHOW {
		suffix := getSuffix()
		return userID + suffix
	}
	return userID
}

func getSuffix() string {
	env := env2.GetEnv()
	parts := strings.Split(env, "-")
	if len(parts) > 0 {
		return "_env" + parts[len(parts)-1]
	}
	return ""
}
