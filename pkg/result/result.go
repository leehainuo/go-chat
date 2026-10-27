package result

import (
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Body 统一响应体。Code 直接采用标准库 net/http 的状态码（设计卡 DR-1 决策：
// 不引入自定义业务错误码分段，全部使用 net/http 自带状态码）。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// ApiError 业务错误。Status 为 net/http 标准状态码，HTTP 响应状态码与 body.code 保持一致。
type ApiError struct {
	Status  int
	Message string
	Cause   error
}

func (e *ApiError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *ApiError) Unwrap() error { return e.Cause }

func New(status int, message string, cause error) *ApiError {
	return &ApiError{Status: status, Message: message, Cause: cause}
}

func Param(message string, cause error) *ApiError {
	return New(http.StatusBadRequest, message, cause)
}

func Unauthorized(message string, cause error) *ApiError {
	return New(http.StatusUnauthorized, message, cause)
}

func Forbidden(message string, cause error) *ApiError {
	return New(http.StatusForbidden, message, cause)
}

func NotFound(message string, cause error) *ApiError {
	return New(http.StatusNotFound, message, cause)
}

func Conflict(message string, cause error) *ApiError {
	return New(http.StatusConflict, message, cause)
}

func TooMany(message string, cause error) *ApiError {
	return New(http.StatusTooManyRequests, message, cause)
}

func System(message string, cause error) *ApiError {
	return New(http.StatusInternalServerError, message, cause)
}

// Write 统一响应出口。遵循设计卡"严禁 panic"约束：业务层只返回 error，
// 全部由本函数转换为统一 JSON；未知错误按 500 处理，不在 handler 中 panic。
func Write(w http.ResponseWriter, r *http.Request, data any, err error) {
	if err != nil {
		var ae *ApiError
		if !errors.As(err, &ae) {
			ae = System("系统繁忙，请稍后再试", err)
		}
		httpx.WriteJsonCtx(r.Context(), w, ae.Status, Body{
			Code:    ae.Status,
			Message: ae.Message,
			Data:    nil,
		})
		return
	}
	httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, Body{
		Code:    http.StatusOK,
		Message: "ok",
		Data:    data,
	})
}
