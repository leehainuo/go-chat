// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-chat/service/user/api/internal/logic/contact"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
)

func GetUnreadApplyCountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetUnreadApplyCountReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := contact.NewGetUnreadApplyCountLogic(r.Context(), svcCtx)
		resp, err := l.GetUnreadApplyCount(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
