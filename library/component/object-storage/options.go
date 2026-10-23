package object_storage

type PutOptions[T any] struct {
	ContentType string
	ACL         string
	// 对应 provider 的选项函数格式为 func(*options)
	ProviderOption func(T)
}

type PutOption[T any] func(*PutOptions[T])
