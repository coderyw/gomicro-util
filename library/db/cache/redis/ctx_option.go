package redis

import "context"

type contextOption struct {
	withoutContext bool
}

type contextOptionKey struct{}

// WithoutTPrefix 执行命令不需要携带租户前缀
//
// Parameters:
//   - ctx context.Context
//
// Returns：
//   - context.Context
func WithoutTPrefix(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	val := ctx.Value(contextOptionKey{})
	if val == nil {
		return context.WithValue(ctx, contextOptionKey{}, &contextOption{withoutContext: true})
	}
	val.(*contextOption).withoutContext = true
	return ctx
}

func getContextOption(ctx context.Context) *contextOption {
	if ctx == nil {
		return nil
	}
	val := ctx.Value(contextOptionKey{})
	if val == nil {
		return nil
	}
	return val.(*contextOption)
}
