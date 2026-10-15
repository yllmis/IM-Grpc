// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/IM_System/apps/im/api/internal/svc"
	"github.com/IM_System/apps/im/api/internal/types"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/useraccess"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetUpUserConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 建立会话
func NewSetUpUserConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetUpUserConversationLogic {
	return &SetUpUserConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetUpUserConversationLogic) SetUpUserConversation(req *types.SetUpUserConversationReq) (resp *types.SetUpUserConversationResp, err error) {
	uid, err := useraccess.RequireUser(l.ctx)
	if err != nil {
		return nil, err
	}
	if req.SendId != "" && req.SendId != uid {
		return nil, useraccess.Denied()
	}
	if _, err := useraccess.ResolveConversation(uid, req.RecvId, "", constants.ChatType(req.ChatType)); err != nil {
		return nil, err
	}
	if constants.ChatType(req.ChatType) == constants.GroupChatType {
		if err := requireGroup(l.ctx, l.svcCtx, uid, req.RecvId); err != nil {
			return nil, err
		}
	}

	_, err = l.svcCtx.SetUpUserConversation(l.ctx, &im.SetUpUserConversationReq{
		SendId:   uid,
		RecvId:   req.RecvId,
		ChatType: req.ChatType,
	})
	return
}
