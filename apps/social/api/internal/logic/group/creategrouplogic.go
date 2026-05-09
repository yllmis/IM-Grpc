// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"github.com/IM_System/apps/im/rpc/imclient"
	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateGroupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创群
func NewCreateGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupLogic {
	return &CreateGroupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateGroupLogic) CreateGroup(req *types.GroupCreateReq) (resp *types.GroupCreateResp, err error) {
	// todo: add your logic here and delete this line

	uid := ctxdata.GetUid(l.ctx)

	// 创建群
	groupResp, err := l.svcCtx.SocialRpc.GroupCreate(l.ctx, &social.GroupCreateReq{
		Name:       req.Name,
		Icon:       req.Icon,
		CreatorUid: uid,
	})
	if err != nil {
		return nil, err
	}

	// 创建群会话
	_, err = l.svcCtx.ImRpc.CreateGroupConversation(l.ctx, &imclient.CreateGroupConversationReq{
		GroupId:  groupResp.Id,
		CreateId: uid,
	})
	return
}
