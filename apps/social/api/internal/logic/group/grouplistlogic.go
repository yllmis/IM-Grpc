// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/jinzhu/copier"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户申群列表
func NewGroupListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupListLogic {
	return &GroupListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 获取用户的群列表
func (l *GroupListLogic) GroupList(req *types.GroupListRep) (resp *types.GroupListResp, err error) {
	// todo: add your logic here and delete this line
	uid := ctxdata.GetUid(l.ctx)
	list, err := l.svcCtx.SocialRpc.GroupList(l.ctx, &social.GroupListReq{
		UserId: uid,
	})

	if err != nil {
		return nil, err
	}

	var respList []*types.Groups
	copier.Copy(&respList, list.Groups)

	return &types.GroupListResp{List: respList}, nil
}
