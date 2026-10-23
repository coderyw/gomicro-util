// Package test
// @Author: yinwei
// @File: make_sql2
// @Version: 1.0.0
// @Date: 2025/3/28 15:31

package test

import (
	"testing"
)

func TestMakeSql(t *testing.T) {
	// outFile, err := os.OpenFile("fix.sql", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	fmt.Println("打开文件失败:", err)
	// 	return
	// }
	// defer outFile.Close()
	// writer := bufio.NewWriter(outFile)
	// var bRecharge = `update ca_balance.b_recharge_%v set status=4,handle_status=3,fail_reason='gcash failed' where third_party_order_id='%v' and user_id=%v;`
	// var bRechargeGidx = `update ca_balance.b_recharge_date_gidx set status=4 where third_party_order_id='%v' and user_id=%v;`
	// var bRechargePartion = `update ca_balance.b_recharge_partition_%v set status=4,handle_status=3,fail_reason='gcash failed' where third_party_order_id='%v' and user_id=%v;`

	// for _, v := range arr {
	// 	is := v[0].(int)
	// 	writer.WriteString(fmt.Sprintf(bRecharge, is%10, v[1], v[0]))
	// 	writer.WriteString("\n")
	// 	writer.WriteString(fmt.Sprintf(bRechargeGidx, v[1], v[0]))
	// 	writer.WriteString("\n")
	// 	writer.WriteString(fmt.Sprintf(bRechargePartion, is%10, v[1], v[0]))
	// 	writer.WriteString("\n")
	// 	writer.Flush()
	// }
}
