package pmicro

import (
	"runtime/debug"

	"go.uber.org/zap"

	"github.com/coderyw/gomicro-util/library/log"
)

// commonUtilModulePath 本库模块路径，用于从依赖列表里定位服务实际编译进去的框架版本
const commonUtilModulePath = "github.com/coderyw/gomicro-util"

// buildInfoFields 从二进制内嵌的构建信息挑白名单字段：go 版本、服务自身的 git 提交/提交时间/是否带未提交改动、
// ca-common-util 依赖版本（replace 到本地目录时另打 replace 目标）。
// 不整体打印 BuildInfo：Settings 里还有 -ldflags / GOFLAGS / CGO_* 这类构建参数，不该随之落日志
func buildInfoFields(bi *debug.BuildInfo) []zap.Field {
	fields := []zap.Field{zap.String("go", bi.GoVersion)}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision", "vcs.time", "vcs.modified":
			fields = append(fields, zap.String(s.Key, s.Value))
		}
	}
	for _, d := range bi.Deps {
		if d.Path != commonUtilModulePath {
			continue
		}
		fields = append(fields, zap.String("common_util", d.Version))
		if d.Replace != nil {
			fields = append(fields, zap.String("common_util_replace", d.Replace.Path+"@"+d.Replace.Version))
		}
	}
	return fields
}

// logBuildInfo 启动时打一条构建信息，运行侧据此确认镜像对应的代码提交与框架版本；
// vcs.* 只有构建上下文带 .git 时才有，Deps 里的版本来自 go.mod / vendor/modules.txt
func logBuildInfo() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Warn("[Init] 二进制未内嵌构建信息")
		return
	}
	log.Info("[Init] common-util构建信息", buildInfoFields(bi)...)
}
