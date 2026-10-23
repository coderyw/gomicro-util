package secret

import "errors"

var (
	// ErrInvalidRef 引用格式不合法
	ErrInvalidRef = errors.New("secret: invalid reference")
	// ErrUnknownScheme 引用的 scheme 没有注册 Provider
	ErrUnknownScheme = errors.New("secret: unknown scheme")
	// ErrKeyNotFound 凭据值中不存在指定 JSON 键
	ErrKeyNotFound = errors.New("secret: key not found in secret")
	// ErrSecretEmpty Provider 返回了空凭据
	ErrSecretEmpty = errors.New("secret: secret is empty")
	// ErrAttemptTimeout 单次拉取超过单次超时（可重试）
	ErrAttemptTimeout = errors.New("secret: fetch attempt timed out")
)

// nonRetryableError 标记为不可重试的 Provider 错误（如凭据不存在、权限拒绝）
type nonRetryableError struct{ err error }

func (e *nonRetryableError) Error() string { return e.err.Error() }
func (e *nonRetryableError) Unwrap() error { return e.err }

// NonRetryable 包装 Provider 侧明确不可重试的错误，避免无意义的退避重试
func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &nonRetryableError{err: err}
}

// IsRetryable 错误是否值得退避重试：NonRetryable 包装过的错误与本包的确定性错误都不重试
func IsRetryable(err error) bool {
	var nr *nonRetryableError
	if errors.As(err, &nr) {
		return false
	}
	return !errors.Is(err, ErrInvalidRef) && !errors.Is(err, ErrUnknownScheme) &&
		!errors.Is(err, ErrKeyNotFound) && !errors.Is(err, ErrSecretEmpty)
}
