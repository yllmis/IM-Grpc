// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupUserListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 成员列表列表
func NewGroupUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupUserListLogic {
	return &GroupUserListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupUserListLogic) GroupUserList(req *types.GroupUserListReq) (resp *types.GroupUserListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
