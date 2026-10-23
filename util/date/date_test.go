// Package date
// @Author: yinwei
// @File: date_test.go
// @Version: 1.0.0
// @Date: 2023/11/2 10:48

package date

import (
	"testing"
)

func TestGetNextMonthYearByRepeat(t *testing.T) {
	type args struct {
		year  int
		month int
		num   int
	}
	tests := []struct {
		name          string
		args          args
		wantNextYear  int
		wantNextMonth int
	}{
		{name: "未超过当年", args: args{
			year:  2023,
			month: 3,
			num:   4,
		}, wantNextYear: 2023, wantNextMonth: 7},
		{name: "超过当年", args: args{
			year:  2023,
			month: 6,
			num:   7,
		}, wantNextYear: 2024, wantNextMonth: 1},
		{name: "按年", args: args{
			year:  2023,
			month: 6,
			num:   12,
		}, wantNextYear: 2024, wantNextMonth: 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNextYear, gotNextMonth := GetNextMonthYearByRepeat(tt.args.year, tt.args.month, tt.args.num)
			if gotNextYear != tt.wantNextYear {
				t.Errorf("GetNextMonthYearByRepeat() gotNextYear = %v, want %v", gotNextYear, tt.wantNextYear)
			}
			if gotNextMonth != tt.wantNextMonth {
				t.Errorf("GetNextMonthYearByRepeat() gotNextMonth = %v, want %v", gotNextMonth, tt.wantNextMonth)
			}
		})
	}
}
