package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/log"
	"gopkg.in/yaml.v3"
)

// fakeProvider 用固定数据模拟凭据后端，并统计拉取次数
type fakeProvider struct {
	scheme  string
	data    map[string]string
	calls   int32
	failN   int32 // 前 N 次返回可重试错误
	fatal   bool  // 返回不可重试错误
	failure error
	delay   time.Duration // 模拟远程耗时，用于并发合并测试
}

func (f *fakeProvider) Scheme() string { return f.scheme }
func (f *fakeProvider) Fetch(_ context.Context, name string) (string, error) {
	n := atomic.AddInt32(&f.calls, 1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.fatal {
		return "", NonRetryable(errors.New("secret not found"))
	}
	if n <= f.failN {
		return "", errors.New("network timeout")
	}
	v, ok := f.data[name]
	if !ok {
		return "", NonRetryable(errors.New("secret not found"))
	}
	return v, nil
}

func setup(t *testing.T, p *fakeProvider) {
	t.Helper()
	ResetCache()
	retryBase = time.Millisecond
	retryCap = 2 * time.Millisecond
	retryDeadline = 50 * time.Millisecond
	retrySleep = func(context.Context, time.Duration) error { return nil }
	Register(p)
	t.Cleanup(func() {
		providerMu.Lock()
		delete(providers, p.scheme)
		rebuildSchemeList()
		providerMu.Unlock()
		ResetCache()
	})
}

type demo struct {
	Password string            `yaml:"password" json:"password"`
	Token    string            `yaml:"token" json:"token"`
	Port     int               `yaml:"port" json:"port"`
	Flag     bool              `yaml:"flag" json:"flag"`
	URL      string            `yaml:"url" json:"url"`
	X        string            `yaml:"x" json:"x"`
	List     []string          `yaml:"list" json:"list"`
	Extra    map[string]string `yaml:"extra" json:"extra"`
}

func TestUnmarshalYAML(t *testing.T) {
	fp := &fakeProvider{scheme: "fakesm", data: map[string]string{
		"ns-svc":   `{"mysql_password":"p@ss#word\"quoted\"\nline2","port":3306,"flag":true}`,
		"ns-plain": `whole-secret-value`,
	}}
	setup(t, fp)
	ctx := context.Background()

	tests := []struct {
		name    string
		in      string
		want    demo
		wantErr error
	}{
		{name: "无引用原样反序列化", in: "password: 'abc'\nport: 1\n", want: demo{Password: "abc", Port: 1}},
		{name: "带键引用（单引号包裹）替换为明文，特殊字符与换行原样保留", in: "password: 'fakesm://ns-svc/mysql_password'",
			want: demo{Password: "p@ss#word\"quoted\"\nline2"}},
		{name: "无键引用取整份凭据", in: "token: fakesm://ns-plain", want: demo{Token: "whole-secret-value"}},
		{name: "非字符串键保留类型", in: "port: fakesm://ns-svc/port\nflag: fakesm://ns-svc/flag", want: demo{Port: 3306, Flag: true}},
		{name: "键不存在报错", in: "x: fakesm://ns-svc/nope", wantErr: ErrKeyNotFound},
		{name: "http 等常见 URL 不被当作引用", in: "url: https://api.example.com/v1/path", want: demo{URL: "https://api.example.com/v1/path"}},
		{name: "未注册 scheme 只透传不报错", in: "x: 'cssm://typo-name/key'", want: demo{X: "cssm://typo-name/key"}},
		{name: "同一文本多处引用逐个替换", in: "token: fakesm://ns-plain\npassword: \"fakesm://ns-svc/mysql_password\"\nport: 1",
			want: demo{Token: "whole-secret-value", Password: "p@ss#word\"quoted\"\nline2", Port: 1}},
		{name: "流式序列与列表项", in: "list: [fakesm://ns-plain, \"fakesm://ns-plain\"]\nextra:\n  k: fakesm://ns-plain\n",
			want: demo{List: []string{"whole-secret-value", "whole-secret-value"}, Extra: map[string]string{"k": "whole-secret-value"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got demo
			err := UnmarshalYAML(ctx, tt.in, &got)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("期望错误 %v，实际 %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("不应报错: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("结果不符\n期望: %+v\n实际: %+v", tt.want, got)
			}
		})
	}
	t.Run("空文本不报错", func(t *testing.T) {
		var got demo
		if err := UnmarshalYAML(ctx, "", &got); err != nil {
			t.Fatal(err)
		}
	})
}

func TestUnmarshalYAML_缓存与重试(t *testing.T) {
	t.Run("同一凭据多次引用只拉取一次", func(t *testing.T) {
		fp := &fakeProvider{scheme: "fakesm", data: map[string]string{"n": `{"a":"1","b":"2"}`}}
		setup(t, fp)
		var out map[string]string
		if err := UnmarshalYAML(context.Background(), "a: fakesm://n/a\nb: fakesm://n/b\nc: fakesm://n", &out); err != nil {
			t.Fatal(err)
		}
		if fp.calls != 1 {
			t.Fatalf("期望拉取 1 次，实际 %d", fp.calls)
		}
	})
	t.Run("可重试错误按次数退避重试后成功", func(t *testing.T) {
		fp := &fakeProvider{scheme: "fakesm", data: map[string]string{"n": "v"}, failN: 2}
		setup(t, fp)
		got, err := Resolve(context.Background(), "fakesm://n")
		if err != nil || got != "v" {
			t.Fatalf("期望重试后成功，实际 got=%q err=%v", got, err)
		}
		if fp.calls != 3 {
			t.Fatalf("期望共调用 3 次，实际 %d", fp.calls)
		}
	})
	t.Run("deadline 内一直失败则放弃并返回最后一次错误", func(t *testing.T) {
		fp := &fakeProvider{scheme: "fakesm", data: map[string]string{"n": "v"}, failN: 1 << 20}
		setup(t, fp)
		retryDeadline = 0 // 不重试：首次失败即返回
		if _, err := Resolve(context.Background(), "fakesm://n"); err == nil || !strings.Contains(err.Error(), "network timeout") {
			t.Fatalf("期望返回最后一次可重试错误，实际 %v", err)
		}
		if fp.calls != 1 {
			t.Fatalf("deadline=0 期望只调用 1 次，实际 %d", fp.calls)
		}
	})
	t.Run("不可重试错误立即返回不重试", func(t *testing.T) {
		fp := &fakeProvider{scheme: "fakesm", fatal: true}
		setup(t, fp)
		if _, err := Resolve(context.Background(), "fakesm://n"); err == nil {
			t.Fatal("期望报错")
		}
		if fp.calls != 1 {
			t.Fatalf("期望只调用 1 次，实际 %d", fp.calls)
		}
	})
	t.Run("空凭据视为错误", func(t *testing.T) {
		fp := &fakeProvider{scheme: "fakesm", data: map[string]string{"n": ""}}
		setup(t, fp)
		if _, err := Resolve(context.Background(), "fakesm://n"); !errors.Is(err, ErrSecretEmpty) {
			t.Fatalf("期望 ErrSecretEmpty，实际 %v", err)
		}
	})
}

func TestResolve_引用格式(t *testing.T) {
	fp := &fakeProvider{scheme: "fakesm", data: map[string]string{"n": "v"}}
	setup(t, fp)
	for _, bad := range []string{"", "fakesm:/n", "'fakesm://n'", "nope://n", "fakesm://n/k/extra", "fakesm://n/a.", "fakesm://-n"} {
		if _, err := Resolve(context.Background(), bad); err == nil {
			t.Errorf("非法引用 %q 应报错", bad)
		}
	}
	if got, err := Resolve(context.Background(), " fakesm://n "); err != nil || got != "v" {
		t.Fatalf("首尾空白应被容忍: got=%q err=%v", got, err)
	}
}

func TestUnmarshalYAML_错误信息不含凭据值(t *testing.T) {
	fp := &fakeProvider{scheme: "fakesm", data: map[string]string{"n": `{"a":"TOPSECRET"}`}}
	setup(t, fp)
	var out map[string]string
	err := UnmarshalYAML(context.Background(), "x: fakesm://n/missing", &out)
	if err == nil || strings.Contains(err.Error(), "TOPSECRET") {
		t.Fatalf("错误信息不得携带凭据明文: %v", err)
	}
}

func TestUnmarshalYAML_注释与整段标量校验(t *testing.T) {
	newFake := func() *fakeProvider {
		return &fakeProvider{scheme: "fakesm", data: map[string]string{"n": `{"a":"1","nul":null}`, "p": "plain"}}
	}
	ctx := context.Background()

	t.Run("注释里的引用不解析也不拉取", func(t *testing.T) {
		fp := newFake()
		setup(t, fp)
		in := "# old: fakesm://n/a\nx: 1 # fakesm://n/a\ny: 'a#b' # fakesm://n/a\n  # fakesm://p\nm: \"a\\\"b\" # fakesm://n/a\ns: 'it''s' # fakesm://p\n"
		var out map[string]interface{}
		if err := UnmarshalYAML(ctx, in, &out); err != nil {
			t.Fatal(err)
		}
		if out["x"] != 1 || out["y"] != "a#b" || out["m"] != `a"b` || out["s"] != "it's" {
			t.Fatalf("解析结果不符: %v", out)
		}
		if fp.calls != 0 {
			t.Fatalf("注释里的引用不应拉取，实际拉取 %d 次", fp.calls)
		}
	})
	t.Run("引号内的 # 不是注释，行尾注释不影响同行引用", func(t *testing.T) {
		setup(t, newFake())
		var out map[string]string
		if err := UnmarshalYAML(ctx, "x: 'fakesm://p' # tail\ny: 'v#1'\n", &out); err != nil || out["x"] != "plain" || out["y"] != "v#1" {
			t.Fatalf("err=%v out=%v", err, out)
		}
	})
	t.Run("块标量、锚点别名、流式集合都按标量语义替换", func(t *testing.T) {
		setup(t, newFake())
		in := "lit: |-\n  fakesm://p\nfold: >-\n  fakesm://p\nanc: &a fakesm://p\nali: *a\nflow: [fakesm://n/a, \"fakesm://n/a\"]\nnext:\n  fakesm://p\n"
		var out struct {
			Lit, Fold, Anc, Ali, Next string
			Flow                      []string
		}
		if err := UnmarshalYAML(ctx, in, &out); err != nil {
			t.Fatal(err)
		}
		for name, v := range map[string]string{"lit": out.Lit, "fold": out.Fold, "anc": out.Anc, "ali": out.Ali, "next": out.Next} {
			if v != "plain" {
				t.Errorf("%s 期望 plain（不带引号），实际 %q", name, v)
			}
		}
		if !reflect.DeepEqual(out.Flow, []string{"1", "1"}) {
			t.Errorf("flow 期望 [1 1]，实际 %v", out.Flow)
		}
	})
	t.Run("拼接在其他内容中的引用判为非法引用且不拉取", func(t *testing.T) {
		fp := newFake()
		setup(t, fp)
		for _, bad := range []string{
			"x: 'fakesm://n/a.'",
			"x: fakesm://n/a/extra",
			"x: foo-fakesm://n/a",
			"u: http://h/p?k=fakesm://n/a",
			"x: 'prefix fakesm://p suffix'",
			"x: \"prefix fakesm://p\"",
			"dsn: mysql://user:fakesm://p@host/db",
			"x: \"line1\n  fakesm://p\"",
			"x: fakesm://n/a;y",
			"x: 124fakesm://p",
			"x: abcfakesm://p",
			"x: \"124fakesm://n/a\"",
		} {
			var out map[string]interface{}
			if err := UnmarshalYAML(ctx, bad, &out); !errors.Is(err, ErrInvalidRef) {
				t.Errorf("%q 期望 ErrInvalidRef，实际 %v", bad, err)
			}
		}
		if fp.calls != 0 {
			t.Fatalf("非法引用不应拉取，实际拉取 %d 次", fp.calls)
		}
	})
	t.Run("映射的键不参与解析", func(t *testing.T) {
		fp := newFake()
		setup(t, fp)
		var out map[string]string
		if err := UnmarshalYAML(ctx, "fakesm://p: v\n", &out); err != nil || out["fakesm://p"] != "v" || fp.calls != 0 {
			t.Fatalf("err=%v out=%v calls=%d", err, out, fp.calls)
		}
	})
	t.Run("键值为 JSON null 视为空凭据", func(t *testing.T) {
		setup(t, newFake())
		var out map[string]string
		if err := UnmarshalYAML(ctx, "x: fakesm://n/nul", &out); !errors.Is(err, ErrSecretEmpty) {
			t.Fatalf("期望 ErrSecretEmpty，实际 %v", err)
		}
	})
	t.Run("引用替换后的类型错误不含凭据明文", func(t *testing.T) {
		setup(t, newFake())
		var out struct {
			X int `yaml:"x"`
		}
		err := UnmarshalYAML(ctx, "x: fakesm://p", &out)
		if err == nil || strings.Contains(err.Error(), "plain") || errors.Unwrap(err) != nil {
			t.Fatalf("期望脱敏后的类型错误，实际 %v", err)
		}
	})
}

func TestUnmarshalJSON(t *testing.T) {
	fp := &fakeProvider{scheme: "fakesm", data: map[string]string{"n": `{"a":"1","port":3306}`, "p": "plain"}}
	setup(t, fp)
	ctx := context.Background()
	t.Run("对象、数组、嵌套中的引用都替换，非字符串键保留类型", func(t *testing.T) {
		var out demo
		in := `{"password":"fakesm://p","port":"fakesm://n/port","list":["fakesm://n/a","x"],"extra":{"k":"fakesm://p"},"url":"https://h/p"}`
		if err := UnmarshalJSON(ctx, in, &out); err != nil {
			t.Fatal(err)
		}
		want := demo{Password: "plain", Port: 3306, List: []string{"1", "x"}, Extra: map[string]string{"k": "plain"}, URL: "https://h/p"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("期望 %+v\n实际 %+v", want, out)
		}
	})
	t.Run("拼接引用判非法", func(t *testing.T) {
		var out demo
		if err := UnmarshalJSON(ctx, `{"x":"a fakesm://p"}`, &out); !errors.Is(err, ErrInvalidRef) {
			t.Fatalf("期望 ErrInvalidRef，实际 %v", err)
		}
	})
	t.Run("非法 json 报错", func(t *testing.T) {
		var out demo
		if err := UnmarshalJSON(ctx, `{"x":`, &out); err == nil {
			t.Fatal("期望报错")
		}
	})
}

func TestUnmarshalYAML_未注册scheme告警去重与忽略名单(t *testing.T) {
	setup(t, &fakeProvider{scheme: "fakesm", data: map[string]string{"n": "v"}})
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		var out map[string]string
		if err := UnmarshalYAML(ctx, "x: cssm://typo/k\ndw: redshift://h:5439/db\nd: dns://svc\nl: lb://svc", &out); err != nil {
			t.Fatal(err)
		}
	}
	if _, warned := warnedSchemes.Load("cssm"); !warned {
		t.Fatal("未注册 scheme 应记录告警")
	}
	for _, s := range []string{"redshift", "dns", "lb"} {
		if _, warned := warnedSchemes.Load(s); warned {
			t.Errorf("忽略名单内的 %s 不应告警", s)
		}
	}
}

func TestUnmarshalYAML_解析出的明文登记到日志脱敏表(t *testing.T) {
	log.ResetRedactValues()
	t.Cleanup(log.ResetRedactValues)
	const whole = `{"mysql_password":"real-db-password","port":3306}`
	setup(t, &fakeProvider{scheme: "fakesm", data: map[string]string{"ns-svc": whole, "ns-plain": "whole-secret-value"}})
	ctx := context.Background()
	text := "a: fakesm://ns-svc/mysql_password\nb: fakesm://ns-plain\nc: fakesm://ns-svc/port"
	assertRegistered := func(t *testing.T) {
		t.Helper()
		for _, plain := range []string{"real-db-password", "whole-secret-value", whole} {
			if got := log.RedactString("x " + plain + " y"); strings.Contains(got, plain) {
				t.Errorf("明文 %q 未登记到脱敏表，RedactString 结果 %q", plain, got)
			}
		}
		if got := log.RedactString("port 3306"); got != "port 3306" {
			t.Errorf("非字符串键不应登记，实际 %q", got)
		}
	}
	t.Run("首次拉取后登记整份凭据与取出的字符串键", func(t *testing.T) {
		var out map[string]interface{}
		if err := UnmarshalYAML(ctx, text, &out); err != nil {
			t.Fatal(err)
		}
		assertRegistered(t)
	})
	t.Run("缓存命中路径同样登记", func(t *testing.T) {
		log.ResetRedactValues()
		var out map[string]interface{}
		if err := UnmarshalYAML(ctx, text, &out); err != nil {
			t.Fatal(err)
		}
		assertRegistered(t)
	})
}

func TestSanitizeDecodeError(t *testing.T) {
	log.ResetRedactValues()
	t.Cleanup(log.ResetRedactValues)
	const secretVal = "CrPw123"
	log.RegisterRedactValue(secretVal)
	t.Run("yaml 类型错误中的凭据原文被抹掉", func(t *testing.T) {
		var target struct {
			Port int `yaml:"port"`
		}
		raw := yaml.Unmarshal([]byte("port: "+secretVal+"\n"), &target)
		if raw == nil || !strings.Contains(raw.Error(), secretVal) {
			t.Fatalf("前置条件：原始 yaml 错误应包含明文，实际 %v", raw)
		}
		got := SanitizeDecodeError(raw)
		if got == nil || strings.Contains(got.Error(), secretVal) {
			t.Fatalf("脱敏后仍包含明文: %v", got)
		}
		if !strings.Contains(got.Error(), "into int") {
			t.Fatalf("应保留类型定位信息，实际 %v", got)
		}
		if errors.Unwrap(got) != nil {
			t.Fatal("不应包装原错误")
		}
	})
	t.Run("未登记的短值：反引号原文仍被整体抹掉", func(t *testing.T) {
		got := SanitizeDecodeError(errors.New("cannot unmarshal !!str `ab` into int"))
		if strings.Contains(got.Error(), "`ab`") {
			t.Fatalf("反引号原文未抹掉: %v", got)
		}
	})
	t.Run("nil 透传", func(t *testing.T) {
		if SanitizeDecodeError(nil) != nil {
			t.Fatal("nil 应原样返回")
		}
	})
}

func TestResolve_并发首次拉取只调用一次Provider(t *testing.T) {
	fp := &fakeProvider{scheme: "fakesm", data: map[string]string{"n": `{"a":"1"}`}, delay: 30 * time.Millisecond}
	setup(t, fp)
	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := Resolve(context.Background(), "fakesm://n/a")
			if err != nil || got != "1" {
				errs <- fmt.Errorf("got=%q err=%v", got, err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	if n := atomic.LoadInt32(&fp.calls); n != 1 {
		t.Fatalf("并发首次拉取应合并为 1 次远程调用，实际 %d 次", n)
	}
	t.Run("拉取失败不写缓存，下次重新拉取", func(t *testing.T) {
		fp2 := &fakeProvider{scheme: "fakesm", data: map[string]string{}, fatal: true}
		setup(t, fp2)
		for i := 0; i < 2; i++ {
			if _, err := Resolve(context.Background(), "fakesm://n"); err == nil {
				t.Fatal("期望报错")
			}
		}
		if fp2.calls != 2 {
			t.Fatalf("失败不应缓存，期望调用 2 次，实际 %d", fp2.calls)
		}
	})
}

// stallProvider 模拟 DEW 响应停滞：Fetch 一直阻塞到 ctx 结束
type stallProvider struct{ calls int32 }

func (s *stallProvider) Scheme() string { return "stallsm" }
func (s *stallProvider) Fetch(ctx context.Context, _ string) (string, error) {
	atomic.AddInt32(&s.calls, 1)
	<-ctx.Done()
	return "", ctx.Err()
}

func TestResolve_凭据后端响应停滞时在deadline内返回(t *testing.T) {
	ResetCache()
	sp := &stallProvider{}
	Register(sp)
	savedBase, savedCap, savedDeadline, savedAttempt, savedSleep := retryBase, retryCap, retryDeadline, retryAttemptTimeout, retrySleep
	t.Cleanup(func() {
		retryBase, retryCap, retryDeadline, retryAttemptTimeout, retrySleep = savedBase, savedCap, savedDeadline, savedAttempt, savedSleep
		providerMu.Lock()
		delete(providers, sp.Scheme())
		rebuildSchemeList()
		providerMu.Unlock()
		ResetCache()
	})
	retryBase, retryCap, retryDeadline, retryAttemptTimeout = time.Millisecond, time.Millisecond, 80*time.Millisecond, 20*time.Millisecond
	retrySleep = func(context.Context, time.Duration) error { return nil }

	start := time.Now()
	_, err := Resolve(context.Background(), "stallsm://ns-svc/mysql_password")
	elapsed := time.Since(start)
	if !errors.Is(err, ErrAttemptTimeout) || !IsRetryable(err) {
		t.Fatalf("期望可重试的单次超时错误，实际 %v", err)
	}
	if elapsed > 80*time.Millisecond+20*time.Millisecond+50*time.Millisecond {
		t.Fatalf("耗时 %v 超出 deadline + 单次超时", elapsed)
	}
	if atomic.LoadInt32(&sp.calls) < 2 {
		t.Fatalf("期望停滞后重试，实际只调用 %d 次", sp.calls)
	}
}

func TestIsIdentifierKey(t *testing.T) {
	cases := map[string]bool{
		"username": true, "user": true, "account": true, "user_name": true,
		"nacos_username": true, "mysql_user": true, "db-account": true, "redis.user": true, "MYSQL_USER": true,
		"password": false, "mysql_password": false, "superuser": false, "user_password": false,
		"token": false, "users_key": false, "": false,
	}
	for k, want := range cases {
		if got := isIdentifierKey(k); got != want {
			t.Errorf("isIdentifierKey(%q) 期望 %v，实际 %v", k, want, got)
		}
	}
}

func TestResolve_账号类键名不登记日志脱敏(t *testing.T) {
	log.ResetRedactValues()
	t.Cleanup(log.ResetRedactValues)
	setup(t, &fakeProvider{scheme: "fakesm", data: map[string]string{
		"ns-test": `{"nacos_username":"nacos-ns-test","nacos_password":"real-nacos-password","mysql_user":"ca_balance"}`,
	}})
	ctx := context.Background()
	for _, ref := range []string{"fakesm://ns-test/nacos_username", "fakesm://ns-test/nacos_password", "fakesm://ns-test/mysql_user"} {
		if _, err := Resolve(ctx, ref); err != nil {
			t.Fatal(err)
		}
	}
	for _, ident := range []string{"nacos-ns-test", "ca_balance"} {
		if got := log.RedactString("db " + ident + " ok"); got != "db "+ident+" ok" {
			t.Errorf("账号类值 %q 不应被打码，实际 %q", ident, got)
		}
	}
	if got := log.RedactString("pwd real-nacos-password"); strings.Contains(got, "real-nacos-password") {
		t.Errorf("密码应被打码，实际 %q", got)
	}
}

func TestUnmarshalJSON_行为对齐json标准库(t *testing.T) {
	setup(t, &fakeProvider{scheme: "fakesm", data: map[string]string{"ns-svc": `{"mysql_password":"real-db-password"}`}})
	ctx := context.Background()
	t.Run("含引用时顶层值后还有内容报错", func(t *testing.T) {
		var out demo
		if err := UnmarshalJSON(ctx, `{"password":"fakesm://ns-svc/mysql_password"} {"x":"1"}`, &out); err == nil {
			t.Fatal("期望报错，实际成功")
		}
	})
	t.Run("顶层值后只有空白不报错", func(t *testing.T) {
		var out demo
		if err := UnmarshalJSON(ctx, "{\"password\":\"fakesm://ns-svc/mysql_password\"}\n  ", &out); err != nil || out.Password != "real-db-password" {
			t.Fatalf("out=%+v err=%v", out, err)
		}
	})
	t.Run("无引用文本直接反序列化，错误类型保留", func(t *testing.T) {
		var out demo
		err := UnmarshalJSON(ctx, `{"port":"not-a-number"}`, &out)
		var te *json.UnmarshalTypeError
		if !errors.As(err, &te) {
			t.Fatalf("期望保留 *json.UnmarshalTypeError，实际 %T %v", err, err)
		}
	})
	t.Run("斜杠转义写法的引用同样解析", func(t *testing.T) {
		var out demo
		if err := UnmarshalJSON(ctx, `{"password":"fakesm:\/\/ns-svc\/mysql_password"}`, &out); err != nil || out.Password != "real-db-password" {
			t.Fatalf("out=%+v err=%v", out, err)
		}
	})
}

func TestUnmarshalYAML_无引用时错误原样返回(t *testing.T) {
	setup(t, &fakeProvider{scheme: "fakesm", data: map[string]string{}})
	var out demo
	err := UnmarshalYAML(context.Background(), "port: abc", &out)
	var te *yaml.TypeError
	if !errors.As(err, &te) || !strings.Contains(err.Error(), "`abc`") {
		t.Fatalf("期望原样返回 *yaml.TypeError 且保留原文，实际 %T %v", err, err)
	}
}

func TestSchemes_返回副本(t *testing.T) {
	setup(t, &fakeProvider{scheme: "fakesm", data: map[string]string{}})
	got := Schemes()
	for i := range got {
		got[i] = "tampered"
	}
	for _, s := range registeredSchemes() {
		if s == "tampered" {
			t.Fatal("修改 Schemes 返回值不应影响内部快照")
		}
	}
}
