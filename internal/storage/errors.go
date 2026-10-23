package storage

import "fmt"

var (
	ErrInvalidArgument  = fmt.Errorf("invalid argument")
	ErrBucketNotFound   = fmt.Errorf("bucket not found")
	ErrInstanceNotFound = fmt.Errorf("instance not found")
	ErrObjectExists     = fmt.Errorf("object already exists")
)

type OpError struct {
	Op       string
	Provider Provider
	Instance string
	Bucket   string
	Key      string
	Err      error
}

func (e *OpError) Error() string {
	return fmt.Sprintf("storage op=%s provider=%s instance=%s bucket=%s key=%s: %v",
		e.Op, e.Provider, e.Instance, e.Bucket, e.Key, e.Err)
}

func (e *OpError) Unwrap() error {
	return e.Err
}
