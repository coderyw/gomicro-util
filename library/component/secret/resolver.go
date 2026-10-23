package secret

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gopkg.in/yaml.v3"
)

// 引用语法：scheme://凭据名[/键名]，必须是标量的全部内容
var refPattern = regexp.MustCompile(`^([a-z][a-z0-9]*)://(` + refName + `)(?:/(` + refName + `))?$`)

// 凭据名 / 键名：字母数字开头结尾，中间允许 _ . -
const refName = `[A-Za-z0-9](?:[A-Za-z0-9_.\-]*[A-Za-z0-9])?`

// 配置里常见、与凭据引用无关的 scheme，扫描时忽略且不告警
var ignoredSchemes = map[string]struct{}{
	"http": {}, "https": {}, "ws": {}, "wss": {}, "grpc": {}, "tcp": {}, "udp": {}, "dns": {}, "lb": {},
	"redis": {}, "rediss": {}, "mysql": {}, "postgres": {}, "postgresql": {}, "mongodb": {}, "redshift": {},
	"amqp": {}, "kafka": {}, "jdbc": {}, "file": {}, "ftp": {}, "sftp": {}, "ssh": {}, "git": {},
	"s3": {}, "obs": {}, "oss": {},
}

// 进程内缓存：key = scheme://name，value = 凭据原文；同一凭据启动期只拉一次
var cache sync.Map

// 同一凭据的并发首次拉取合并为一次远程调用，其余调用方等待同一结果
var fetchGroup singleflight.Group

// 未注册 scheme 每个进程只告警一次：deep link 之类的存量值每次加载配置都会命中，逐次告警只是噪音
var warnedSchemes sync.Map

// ResetCache 清空进程内缓存与告警去重记录（测试用）
func ResetCache() {
	cache = sync.Map{}
	warnedSchemes = sync.Map{}
}

// IsRef 判断值整体是否为已注册 scheme 的凭据引用：忽略名单（http/mysql 等）与未注册 scheme 均返回 false 且不告警。
// 供启动 env（如 NACOS_USERNAME）这类不走 yaml 解析的单值做"引用还是明文"判别
func IsRef(value string) bool {
	m := refPattern.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return false
	}
	if _, ignored := ignoredSchemes[m[1]]; ignored {
		return false
	}
	_, err := getProvider(m[1])
	return err == nil
}

// Resolve 解析单个引用，返回明文（无键名 = 整份凭据；有键名 = JSON 中取该键）
func Resolve(ctx context.Context, ref string) (string, error) {
	m := refPattern.FindStringSubmatch(strings.TrimSpace(ref))
	if m == nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidRef, ref)
	}
	val, _, err := resolveParts(ctx, m[1], m[2], m[3])
	return val, err
}

// UnmarshalYAML 解析 yaml 文本中的凭据引用后反序列化到 out。
// 走 yaml 解析器的节点树：只看标量节点的值，注释、键名、锚点/别名、块标量、引号风格都由解析器处理；
// 标量值整体是引用才替换，引用被拼在其他内容里（DSN/URL/带前后缀文字）判 ErrInvalidRef；
// 无引用的文本与直接 yaml.Unmarshal 等价、错误原样返回；替换过引用时反序列化错误经 SanitizeDecodeError 脱去凭据值
func UnmarshalYAML(ctx context.Context, text string, out interface{}) error {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(text), &root); err != nil {
		return err // 尚未解析引用，文本里只有引用没有明文
	}
	if root.Kind == 0 {
		return nil // 空文档
	}
	replaced := 0
	if err := resolveYAMLNode(ctx, &root, &replaced); err != nil {
		return err
	}
	if replaced == 0 {
		return root.Decode(out)
	}
	return SanitizeDecodeError(root.Decode(out))
}

// UnmarshalJSON 解析 json 文本中的凭据引用后反序列化到 out（语义同 UnmarshalYAML）。
// 文本里不含任何已注册 scheme 时直接 json.Unmarshal，与改造前行为与开销完全一致；
// 含引用时先解析成通用树替换引用，再序列化后反序列化到 out
func UnmarshalJSON(ctx context.Context, text string, out interface{}) error {
	if !mayContainRef(text) {
		return json.Unmarshal([]byte(text), out)
	}
	var doc interface{}
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return err // 尚未解析引用，文本里只有引用没有明文
	}
	// Decoder 只读第一个值，顶层值之后还有内容时 json.Unmarshal 会报错，这里对齐
	if _, err := dec.Token(); err != io.EOF {
		return errTrailingJSON
	}
	replaced := 0
	resolved, err := resolveJSONValue(ctx, doc, &replaced)
	if err != nil {
		return err
	}
	if replaced == 0 {
		return json.Unmarshal([]byte(text), out)
	}
	buf, err := json.Marshal(resolved)
	if err != nil {
		return SanitizeDecodeError(err)
	}
	return SanitizeDecodeError(json.Unmarshal(buf, out))
}

var errTrailingJSON = errors.New("invalid JSON: unexpected content after top-level value")

// mayContainRef 文本里是否可能有凭据引用：按 "scheme:" 而不是 "scheme://" 判定，
// 覆盖 JSON 把 / 转义成 \/ 的写法（csms:\/\/...），宁可多走一次完整解析也不漏判
func mayContainRef(text string) bool {
	for _, sch := range registeredSchemes() {
		if strings.Contains(text, sch+":") {
			return true
		}
	}
	return false
}

// resolveYAMLNode 遍历节点树，替换值位置上的引用标量；映射的键不参与。replaced 累计替换个数
func resolveYAMLNode(ctx context.Context, n *yaml.Node, replaced *int) error {
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			if err := resolveYAMLNode(ctx, c, replaced); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		for i := 1; i < len(n.Content); i += 2 {
			if err := resolveYAMLNode(ctx, n.Content[i], replaced); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		return resolveYAMLScalar(ctx, n, replaced)
	case yaml.AliasNode:
		// 别名指向的锚点节点已在树中遍历过，这里不重复处理
	}
	return nil
}

func resolveYAMLScalar(ctx context.Context, n *yaml.Node, replaced *int) error {
	scheme, name, key, ok, err := classifyScalar(n.Value)
	if err != nil {
		return fmt.Errorf("%w (line %d)", err, n.Line)
	}
	if !ok {
		return nil
	}
	val, isString, err := resolveParts(ctx, scheme, name, key)
	if err != nil {
		return err
	}
	n.Value = val
	*replaced++
	if isString {
		n.Tag, n.Style = "!!str", yaml.DoubleQuotedStyle // 明文按字符串落地，不再参与 yaml 类型推断
	} else {
		n.Tag, n.Style = "", 0 // 数字/布尔按原始 JSON 文本交给 yaml 推断类型
	}
	return nil
}

// resolveJSONValue 递归替换 json 文档里值位置上的引用字符串。replaced 累计替换个数
func resolveJSONValue(ctx context.Context, v interface{}, replaced *int) (interface{}, error) {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, c := range t {
			r, err := resolveJSONValue(ctx, c, replaced)
			if err != nil {
				return nil, err
			}
			t[k] = r
		}
		return t, nil
	case []interface{}:
		for i, c := range t {
			r, err := resolveJSONValue(ctx, c, replaced)
			if err != nil {
				return nil, err
			}
			t[i] = r
		}
		return t, nil
	case string:
		scheme, name, key, ok, err := classifyScalar(t)
		if err != nil || !ok {
			return v, err
		}
		val, isString, err := resolveParts(ctx, scheme, name, key)
		if err != nil {
			return nil, err
		}
		*replaced++
		if isString {
			return val, nil
		}
		return json.RawMessage(val), nil
	}
	return v, nil
}

// classifyScalar 判断一个标量值是否为需要解析的凭据引用：
// 整体是已注册 scheme 的引用 → ok；值里任意位置含已注册 scheme 的 "xxx://" 但不是整体 → ErrInvalidRef；
// 忽略名单 scheme 或未注册 scheme → 不处理（未注册只告警一次）
func classifyScalar(value string) (scheme, name, key string, ok bool, err error) {
	unknown := ""
	if m := refPattern.FindStringSubmatch(value); m != nil {
		if _, ignored := ignoredSchemes[m[1]]; ignored {
			return "", "", "", false, nil
		}
		if _, e := getProvider(m[1]); e == nil {
			return m[1], m[2], m[3], true, nil
		}
		unknown = m[1] // 整体像引用但 scheme 未注册：先做子串检查，再决定是否只告警透传
	}
	// 值里任意位置出现已注册 scheme 的 "xxx://" 即视为引用被拼在其他内容里：
	// 按子串判定而不按单词切分，`124csms://x`、`abccsms://x` 这类紧贴前缀的写法同样拦下，否则会静默变成字面量密码
	for _, sch := range registeredSchemes() {
		if strings.Contains(value, sch+"://") {
			return "", "", "", false, fmt.Errorf("%w: %s:// reference must be the whole scalar value, embedding in DSN/URL/text is not supported", ErrInvalidRef, sch)
		}
	}
	if unknown != "" {
		warnUnknownScheme(unknown)
	}
	return "", "", "", false, nil
}

// warnUnknownScheme 长得像引用但 scheme 未注册：多半是拼写错误，只告警不替换（不打印值），同一 scheme 只告警一次
func warnUnknownScheme(scheme string) {
	if _, dup := warnedSchemes.LoadOrStore(scheme, struct{}{}); dup {
		return
	}
	log.Warn("[secret] 疑似凭据引用的 scheme 未注册，原样透传", zap.String("scheme", scheme))
}

// resolveParts 返回凭据值及其是否为字符串（决定注入时是否按字符串落地）
func resolveParts(ctx context.Context, scheme, name, key string) (string, bool, error) {
	p, err := getProvider(scheme)
	if err != nil {
		return "", false, err
	}
	cacheKey := scheme + "://" + name
	raw, ok := cache.Load(cacheKey)
	if !ok {
		raw, err, _ = fetchGroup.Do(cacheKey, func() (interface{}, error) {
			if v, hit := cache.Load(cacheKey); hit {
				return v, nil // 等待期间已被同 key 的前一次调用写入缓存
			}
			var fetched string
			if e := retry(ctx, cacheKey, func(actx context.Context) error {
				var e error
				fetched, e = p.Fetch(actx, name)
				return e
			}); e != nil {
				log.Error("[secret] 拉取凭据失败", zap.String("scheme", scheme), zap.String("name", name), zap.Error(e))
				return nil, fmt.Errorf("secret %s://%s: %w", scheme, name, e)
			}
			if fetched == "" {
				return nil, fmt.Errorf("secret %s://%s: %w", scheme, name, ErrSecretEmpty)
			}
			cache.Store(cacheKey, fetched)
			return fetched, nil
		})
		if err != nil {
			return "", false, err
		}
	}
	secretStr := raw.(string)
	// 解析出的明文登记到日志脱敏表：无论后续被谁打进日志，出现即替换为掩码（按值判定，缓存命中路径同样登记）。
	// 整份凭据原文总是登记；按键取出的字符串值除标识类键名外登记，见 isIdentifierKey
	log.RegisterRedactValue(secretStr)
	if key == "" {
		return secretStr, true, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(secretStr), &obj); err != nil {
		return "", false, fmt.Errorf("secret %s://%s: value is not a JSON object, key %q unavailable", scheme, name, key)
	}
	rawVal, ok := obj[key]
	if !ok {
		return "", false, fmt.Errorf("secret %s://%s/%s: %w", scheme, name, key, ErrKeyNotFound)
	}
	if bytes.Equal(bytes.TrimSpace(rawVal), []byte("null")) {
		// JSON null 反序列化到 string 不报错会静默注入空串，绕过空凭据的错误语义，这里显式拦下
		return "", false, fmt.Errorf("secret %s://%s/%s: %w", scheme, name, key, ErrSecretEmpty)
	}
	var s string
	if err := json.Unmarshal(rawVal, &s); err == nil {
		if !isIdentifierKey(key) {
			log.RegisterRedactValue(s)
		}
		return s, true, nil
	}
	return string(rawVal), false, nil // 非字符串键（数字/布尔）按原始 JSON 文本返回
}

// 标识类键名后缀：账号名不是秘密，却常与库名、服务名、租户名同串（如 ca_balance、nacos-ns-test），
// 登记脱敏会让这些词在所有日志里被打码。按"键名等于后缀，或以分隔符 _ - . 加后缀结尾"判定，不区分大小写
var identifierKeySuffixes = []string{"username", "user_name", "user", "account"}

// isIdentifierKey 键名是否为标识类（不登记日志脱敏）。只对明确的账号类键名放行，未命中的键一律登记：
// 判定失误的代价只是多打码，不会让真正的秘密漏出脱敏
func isIdentifierKey(key string) bool {
	k := strings.ToLower(key)
	for _, suf := range identifierKeySuffixes {
		if k == suf {
			return true
		}
		if len(k) > len(suf) && strings.HasSuffix(k, suf) {
			switch k[len(k)-len(suf)-1] {
			case '_', '-', '.':
				return true
			}
		}
	}
	return false
}
