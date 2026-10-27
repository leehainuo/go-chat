// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-chat/service/user/api/internal/logic/user"
	"go-chat/service/user/api/internal/svc"
)

func RefreshWsServerUriHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := user.NewRefreshWsServerUriLogic(r.Context(), svcCtx)
		resp, err := l.RefreshWsServerUri()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
