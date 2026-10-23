// Package date 日期处理工具包
// @Author: yinwei
// @File: date
// @Version: 1.0.0
// @Date: 2023/11/2 10:22
//
// 日期处理工具包，这里的日期都与标准库里的一致

package date

import "time"

var (
	// day30 有30天的月份集合
	// 包括：4月(April)、6月(June)、9月(September)、11月(November)
	day30 = map[time.Month]struct{}{
		time.April:     struct{}{},
		time.June:      struct{}{},
		time.September: struct{}{},
		time.November:  struct{}{},
	}

	// day31 有31天的月份集合
	// 包括：1月、3月、5月、7月、8月、10月、12月
	day31 = map[time.Month]struct{}{
		time.January:  struct{}{},
		time.March:    struct{}{},
		time.May:      struct{}{},
		time.July:     struct{}{},
		time.August:   struct{}{},
		time.October:  struct{}{},
		time.December: struct{}{},
	}
)

// GetMonthDayNum 获取指定年月中该月份有多少天
// 参数:
//   - year: 年份，用于判断闰年
//   - month: 月份 (1-12)
//
// 返回值:
//   - int: 该月份的天数 (28, 29, 30, 或 31)
func GetMonthDayNum(year int, month int) int {
	// 检查是否为31天的月份
	if _, ok := day31[time.Month(month)]; ok {
		return 31
	}

	// 检查是否为30天的月份
	if _, ok := day30[time.Month(month)]; ok {
		return 30
	}

	// 剩下的就是2月，需要判断是平年还是闰年
	// 闰年判断规则：能被4整除但不能被100整除，或者能被400整除
	if (year%4 == 0 && year%100 != 0) || year%400 == 0 {
		// 闰年2月有29天
		return 29
	}
	// 平年2月有28天
	return 28
}

// GetNextMonthYearByRepeat 根据给定的年月，通过在月份上累加指定数值，计算返回正确的年份和月份
// 用于处理月份跨年的情况，例如从11月加3个月会到次年2月
// 参数:
//   - year: 起始年份
//   - month: 起始月份 (1-12)
//   - num: 要累加的月份数
//
// 返回值:
//   - nextYear: 计算后的年份
//   - nextMonth: 计算后的月份 (1-12)
func GetNextMonthYearByRepeat(year, month, num int) (nextYear int, nextMonth int) {
	nextYear, nextMonth = year, month
	nextMonth += num

	// 如果月份正好是12，直接返回
	if nextMonth == 12 {
		return
	}

	// 计算跨年情况
	nextYear = nextYear + nextMonth/12 // 根据月份总数计算需要增加的年份
	nextMonth = nextMonth % 12         // 获取最终的月份 (取余操作)

	// 处理月份为0的情况 (12月的情况)
	if nextMonth == 0 {
		nextMonth = 12
		nextYear--
	}

	return
}
