// Package gin gin 参数校验与统一响应封装。
//
// 配合 pmicro 的 option.RegisterHttpHandler(f, true) 使用默认响应结构（model/http.NsHttpResp）：
//
//	func Handler(c *gin.Context) {
//		req := new(Req)
//		if err := gin.BindAndCheck(c, req); err != nil {
//			gin.SetError(c, err) // 由中间件解析错误码并返回
//			return
//		}
//		gin.SetResponse(c, data) // 包装为 {code, msg, detail, data}
//	}
//
// 不需要默认响应结构时调用 SetNoDefaultResponse 后自行写响应体。
//
// 文档：docs/components/http.md
package gin
