package config

import "testing"

func TestPoolStatsMode(t *testing.T) {
	cases := []struct {
		mode   int
		log    bool
		report bool
	}{
		{PoolStatsModeOff, false, false},
		{PoolStatsModeDefault, true, false}, // 0 默认等同于 1：仅日志
		{PoolStatsModeLog, true, false},
		{PoolStatsModeReport, false, true},
		{PoolStatsModeBoth, true, true},
	}
	for _, c := range cases {
		if got := PoolStatsLogEnabled(c.mode); got != c.log {
			t.Errorf("PoolStatsLogEnabled(%d) = %v, want %v", c.mode, got, c.log)
		}
		if got := PoolStatsReportEnabled(c.mode); got != c.report {
			t.Errorf("PoolStatsReportEnabled(%d) = %v, want %v", c.mode, got, c.report)
		}
	}
}
