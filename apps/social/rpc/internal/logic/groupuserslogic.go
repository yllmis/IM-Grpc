package logic

import (
	"context"

	"github.com/IM_System/apps/social/rpc/internal/svc"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/xerr"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupusersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGroupusersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupusersLogic {
	return &GroupusersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GroupusersLogic) Groupusers(in *social.GroupusersReq) (*social.GroupusersResp, error) {
	// todo: add your logic here and delete this line

	groupMembers, err := l.svcCtx.GroupMembersModel.ListByGroupId(l.ctx, in.GroupId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "list group member err %v req %v", err, in.GroupId)
	}

	var respList []*social.GroupMembers
	copier.Copy(&respList, &groupMembers)

	return &social.GroupusersResp{
		List: respList,
	}, nil
}
