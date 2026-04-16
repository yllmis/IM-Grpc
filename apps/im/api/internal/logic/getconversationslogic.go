// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/IM_System/apps/im/api/internal/svc"
	"github.com/IM_System/apps/im/api/internal/types"
	"github.com/IM_System/apps/im/rpc/imclient"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/jinzhu/copier"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取会话
func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationsLogic) GetConversations(req *types.GetConversationsReq) (resp *types.GetConversationsResp, err error) {
	// todo: add your logic here and delete this line

	uid := ctxdata.GetUid(l.ctx)

	data, err := l.svcCtx.GetConversations(l.ctx, &imclient.GetConversationsReq{
		UserId: uid,
	})

	if err != nil {
		return nil, err
	}

	var res types.GetConversationsResp
	copier.Copy(&res, data)
	return &res, nil
}
