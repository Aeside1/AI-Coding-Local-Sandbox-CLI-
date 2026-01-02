package errors

import "errors"

// 预定义错误
var (
	ErrPathEscape      = errors.New("路径越界：禁止访问工作目录外的文件")
	ErrAbsolutePath    = errors.New("安全限制：禁止使用绝对路径")
	ErrParentDir       = errors.New("安全限制：禁止访问父目录")
	ErrComposeNotFound = errors.New("docker-compose.yml 文件不存在")
	ErrDockerNotReady  = errors.New("Docker 未运行或不可用")
)

// WrapError 包装错误
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return &wrappedError{
		msg: message,
		err: err,
	}
}

type wrappedError struct {
	msg string
	err error
}

func (e *wrappedError) Error() string {
	return e.msg + ": " + e.err.Error()
}

func (e *wrappedError) Unwrap() error {
	return e.err
}
