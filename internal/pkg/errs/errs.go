package errs

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string { return e.Message }

func New(code int, message string) *Error { return &Error{Code: code, Message: message} }

var (
	ErrParam          = New(1001, "参数错误")
	ErrUnauthorized   = New(2001, "未登录或登录已过期")
	ErrWrongPassword  = New(2002, "用户名或密码错误")
	ErrUsernameExists = New(2003, "用户名已存在")
	ErrUserNotFound   = New(2004, "用户不存在")
	ErrNoteNotFound   = New(3001, "笔记不存在")
	ErrForbidden      = New(3003, "无权操作")
	ErrInternal       = New(5000, "服务器内部错误")
)
