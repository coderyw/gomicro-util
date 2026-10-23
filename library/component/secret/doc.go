// Package secret 凭据引用解析组件。
//
// 配置文本中的敏感值不再明文存放，而是写成凭据引用：
//
//	{scheme}://{凭据名}[/{JSON键名}]
//	csms://ns-ca-balance/mysql_password    华为云 CSMS
//	awssm://ok-ca-balance/mysql_password   AWS Secrets Manager
//
// 组件在配置反序列化时把引用替换成凭据明文（UnmarshalYAML / UnmarshalJSON），按 scheme 路由到对应 Provider。
// 替换走 yaml / json 解析器的节点树：只看值位置上的标量，引用必须是标量的全部内容（块标量、引号风格、锚点别名、
// 流式集合都由解析器处理），拼在 DSN/URL/文字中间的引用判非法；注释和映射键不参与。无引用的文本与直接反序列化等价。
// 本次替换过引用时，反序列化错误经 SanitizeDecodeError 脱去凭据值再返回；没有引用的文本错误原样返回。
// 值里任意位置出现已注册 scheme 的 "xxx://" 即视为引用：普通配置项（文案、链接）不能含 csms:// / awssm:// 子串，
// 否则整份配置解析失败——启动期直接退出，热更新期只打 ERROR 并保留旧配置。
// 凭据在进程内只拉取一次并缓存，轮换靠重启生效。单次拉取有超时，失败按分类重试并以整体 deadline 收口（见 retry.go）。
// 解析出的字符串明文登记到 log 包的脱敏表（按值判定），此后出现在经 log 包输出的日志里都被替换为掩码，与按字段名的脱敏叠加；
// 键名为账号类（username / user / account 等，见 isIdentifierKey）的值不登记，避免账号名与库名、服务名同串时被大面积打码。
// 只把秘密写成引用：库名、主机、端口等非秘密配置不要放进凭据，它们一旦被引用，取出的值同样会登记脱敏。
package secret
