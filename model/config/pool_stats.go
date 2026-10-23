/**
 * @Author: yinwei
 * @Description: 连接池统计模式配置（mysql/redis 共用）
 * @File: pool_stats
 * @Version: 1.0.0
 * @Date: 2026/8/20 10:00
 */
package config

// 连接池统计模式取值：
//   - -1 不统计：既不打印日志也不上报指标
//   - 0  默认值：等同于 1（仅日志打印）
//   - 1  仅开启日志打印
//   - 2  仅开启指标上报（推送到 pmicro/metrics，metrics 未开启时数据被丢弃）
//   - 3  日志打印和指标上报都开启
const (
	PoolStatsModeOff     = -1
	PoolStatsModeDefault = 0
	PoolStatsModeLog     = 1
	PoolStatsModeReport  = 2
	PoolStatsModeBoth    = 3
)

// normalizePoolStatsMode 把默认值 0 归一化为默认行为（仅日志打印）
func normalizePoolStatsMode(mode int) int {
	if mode == PoolStatsModeDefault {
		return PoolStatsModeLog
	}
	return mode
}

// PoolStatsLogEnabled 该模式下是否打印连接池统计日志
func PoolStatsLogEnabled(mode int) bool {
	m := normalizePoolStatsMode(mode)
	return m == PoolStatsModeLog || m == PoolStatsModeBoth
}

// PoolStatsReportEnabled 该模式下是否上报连接池统计指标
func PoolStatsReportEnabled(mode int) bool {
	m := normalizePoolStatsMode(mode)
	return m == PoolStatsModeReport || m == PoolStatsModeBoth
}
