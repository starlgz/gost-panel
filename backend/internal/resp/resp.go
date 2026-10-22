// Package resp 提供统一的 API 响应格式与辅助函数。
//
// 所有接口统一返回：
//
//	{"code": 0, "message": "ok", "data": ...}   // 成功
//	{"code": 40001, "message": "...", "data": null} // 失败
//
// code 规则：0 = 成功；非 0 = 失败，且与 HTTP 状态码对应（业务码 = HTTP码*100 + 序号）。
// message 对应前端提示文案；data 为业务数据（成功时），失败时为 null。
package resp

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body 统一响应体。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// 预定义业务码（HTTP 状态 = code/100）。
const (
	CodeOK           = 0
	CodeInvalidArgs  = 42200 // 参数错误 (HTTP 422)
	CodeBadRequest   = 40000 // 一般业务错误 (HTTP 400)
	CodeUnauthorized = 40100 // 未认证 (HTTP 401)
	CodeNotFound     = 40400 // 资源不存在 (HTTP 404)
	CodeServerError  = 50000 // 服务器内部错误 (HTTP 500)
	CodeUpstream     = 50200 // 上游（gost）错误 (HTTP 502)
)

// ErrNotFound 供 store 层返回、API 层用 errors.Is 判断。
var ErrNotFound = errors.New("记录不存在")

// OK 返回成功响应，data 可为 nil。
func OK(c *gin.Context, data any) {
	if data == nil {
		data = gin.H{"ok": true}
	}
	c.JSON(http.StatusOK, Body{Code: CodeOK, Message: "ok", Data: data})
}

// OKMsg 返回带提示消息的成功响应。
func OKMsg(c *gin.Context, message string, data any) {
	if data == nil {
		data = gin.H{"ok": true}
	}
	c.JSON(http.StatusOK, Body{Code: CodeOK, Message: message, Data: data})
}

// Err 返回失败响应：httpCode 为 HTTP 状态码，bizCode 为业务码，msg 为提示文案。
func Err(c *gin.Context, httpCode, bizCode int, msg string) {
	c.JSON(httpCode, Body{Code: bizCode, Message: msg, Data: nil})
}

// 便捷封装：覆盖本项目全部失败场景。

// ErrArgs 参数校验失败 (HTTP 422)。
func ErrArgs(c *gin.Context, msg string) {
	Err(c, http.StatusUnprocessableEntity, CodeInvalidArgs, msg)
}

// ErrBadRequest 业务规则错误，如端口冲突 (HTTP 400)。
func ErrBadRequest(c *gin.Context, msg string) {
	Err(c, http.StatusBadRequest, CodeBadRequest, msg)
}

// ErrUnauthorized 认证失败 (HTTP 401)。
func ErrUnauthorized(c *gin.Context, msg string) {
	Err(c, http.StatusUnauthorized, CodeUnauthorized, msg)
}

// ErrNotFoundResp 资源不存在 (HTTP 404)。
func ErrNotFoundResp(c *gin.Context, msg string) {
	Err(c, http.StatusNotFound, CodeNotFound, msg)
}

// ErrInternal 服务器内部错误 (HTTP 500)。
func ErrInternal(c *gin.Context, msg string) {
	Err(c, http.StatusInternalServerError, CodeServerError, msg)
}

// ErrUpstream 上游 gost 错误 (HTTP 502)。
func ErrUpstream(c *gin.Context, msg string) {
	Err(c, http.StatusBadGateway, CodeUpstream, msg)
}
