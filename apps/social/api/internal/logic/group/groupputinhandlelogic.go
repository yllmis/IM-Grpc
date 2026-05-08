// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"github.com/IM_System/apps/im/rpc/imclient"
	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupPutInHandleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 申请进群处理
func NewGroupPutInHandleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupPutInHandleLogic {
	return &GroupPutInHandleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupPutInHandleLogic) GroupPutInHandle(req *types.GroupPutInHandleRep) (resp *types.GroupPutInHandleResp, err error) {

	handleResp, err := l.svcCtx.SocialRpc.GroupPutInHandle(l.ctx, &social.GroupPutInHandleReq{
		GroupReqId:   req.GroupReqId,
		GroupId:      req.GroupId,
		HandleUid:    ctxdata.GetUid(l.ctx),
		HandleResult: req.HandleResult,
	})
	if err != nil {
		return nil, err
	}

	if constants.HandlerResult(req.HandleResult) != constants.PassHandlerResult {
		return
	}

	// step1：创建申请人和群的会话
	_, err = l.svcCtx.ImRpc.SetUpUserConversation(l.ctx, &imclient.SetUpUserConversationReq{
		SendId:   handleResp.ReqId,
		RecvId:   handleResp.GroupId,
		ChatType: int32(constants.GroupChatType),
	})
	if err != nil {
		logx.Errorf("ImRpc.SetUpUserConversation failed: %v", err)
	}

	return
}
