// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"

	"go-chat/pkg/result"
	"go-chat/service/user/api/internal/config"
	"go-chat/service/user/api/internal/handler"
	"go-chat/service/user/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/user_api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 统一响应契约（DR-1 决策）：成功固定 {code:200,message:"ok",data}；
	// 错误使用标准库 net/http 状态码，HTTP 状态码与 body.code 一致；严禁 panic，错误统一经此出口。
	httpx.SetOkHandler(func(ctx context.Context, v any) any {
		return result.Body{Code: http.StatusOK, Message: "ok", Data: v}
	})
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, any) {
		var ae *result.ApiError
		if errors.As(err, &ae) {
			return ae.Status, result.Body{Code: ae.Status, Message: ae.Message}
		}
		logx.WithContext(ctx).Errorf("internal error: %+v", err)
		return http.StatusInternalServerError, result.Body{
			Code:    http.StatusInternalServerError,
			Message: "系统繁忙，请稍后再试",
		}
	})

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
