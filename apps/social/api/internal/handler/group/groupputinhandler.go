// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"net/http"

	"github.com/IM_System/apps/social/api/internal/logic/group"
	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 申请进群
func GroupPutInHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GroupPutInRep
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := group.NewGroupPutInLogic(r.Context(), svcCtx)
		resp, err := l.GroupPutIn(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
