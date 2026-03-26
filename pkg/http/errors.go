package http

import (
	"errors"
)

// 错误定义
var (
	ErrBadRequest      = errors.New("400 Bad Request")
	ErrUnauthorized    = errors.New("401 Unauthorized")
	ErrForbidden       = errors.New("403 Forbidden")
	ErrNotFound        = errors.New("404 Not Found")
	ErrMethodNotAllowed = errors.New("405 Method Not Allowed")
	ErrInternalServer  = errors.New("500 Internal Server Error")
	ErrBadGateway      = errors.New("502 Bad Gateway")
	ErrServiceUnavailable = errors.New("503 Service Unavailable")
	ErrGatewayTimeout  = errors.New("504 Gateway Timeout")
)