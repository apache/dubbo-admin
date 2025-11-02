package bizerror

type Error interface {
	Code() ErrorCode
	Message() string
	Error() string
}

type ErrorCode string

const (
	UnknownError    ErrorCode = "UnknownError"
	InvalidArgument ErrorCode = "InvalidArgument"
	StoreError      ErrorCode = "StoreError"
	AppNotFound     ErrorCode = "AppNotFound"
	Unauthorized    ErrorCode = "Unauthorized"
	SessionError    ErrorCode = "SessionError"
)

type bizError struct {
	code    ErrorCode
	message string
}

var _ Error = &bizError{}

func NewBizError(code ErrorCode, message string) Error {
	return &bizError{
		code:    code,
		message: message,
	}
}

func (b *bizError) Code() ErrorCode {
	return b.code
}

func (b *bizError) Message() string {
	return b.message
}

func (b *bizError) Error() string {
	return string(b.code) + ": " + b.message
}
