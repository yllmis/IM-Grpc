package logic

import (
	"context"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/im/rpc/internal/svc"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/wuid"
	"github.com/IM_System/pkg/xerr"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetUpUserConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetUpUserConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetUpUserConversationLogic {
	return &SetUpUserConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 建立会话: 群聊, 私聊
func (l *SetUpUserConversationLogic) SetUpUserConversation(in *im.SetUpUserConversationReq) (*im.SetUpUserConversationResp, error) {
	// todo: add your logic here and delete this line

	switch constants.ChatType(in.ChatType) {
	case constants.SingleChatType:
		// 私聊，建立双方的会话

		// 生成会话的id
		conversationId := wuid.CombineId(in.SendId, in.RecvId)
		// 验证是否建立过
		conversationRes, err := l.svcCtx.ConversationModel.FindOne(l.ctx, conversationId)
		if err != nil {
			// 没有建立过，建立会话
			if err == immodels.ErrNotFound {
				err = l.svcCtx.ConversationModel.Insert(l.ctx, &immodels.Conversation{
					ConversationId: conversationId,
					ChatType:       constants.SingleChatType,
				})
				if err != nil {
					return nil, errors.Wrapf(xerr.NewDBErr(), "Conversation.Insert err %v", err)
				}
			} else {
				return nil, errors.Wrapf(xerr.NewDBErr(), "Conversation.FindOne err %v, req %v", err, conversationId)
			}
		} else if conversationRes != nil {
			// 已经建立过会话主记录，继续确保双方用户会话列表存在
		}
		// 建立双方的会话
		err = l.setUpUserConversation(conversationId, in.SendId, in.RecvId, constants.SingleChatType, true)
		if err != nil {
			return nil, err
		}
		err = l.setUpUserConversation(conversationId, in.RecvId, in.SendId, constants.SingleChatType, false)
		if err != nil {
			return nil, err
		}
	case constants.GroupChatType:
		// 群聊，建立群成员的会话
		// 群id，创建者id，群id
		err := l.setUpUserConversation(in.RecvId, in.SendId, in.RecvId, constants.GroupChatType, true)
		if err != nil {
			return nil, err
		}
	}

	return &im.SetUpUserConversationResp{}, nil
}

func (l *SetUpUserConversationLogic) setUpUserConversation(conversationId, userId, recvId string, chatType constants.ChatType, isShow bool) error {
	// 用户的会话列表
	conversations, err := l.svcCtx.ConversationsModel.FindByUserId(l.ctx, userId)
	if err != nil {
		if err == immodels.ErrNotFound {
			// 没有会话列表，创建会话列表
			conversations = &immodels.Conversations{
				ID:               bson.NewObjectID(),
				UserId:           userId,
				ConversationList: make(map[string]*immodels.Conversation),
			}
		} else {
			return errors.Wrapf(xerr.NewDBErr(), "ConversationsModel.FindByUserId err %v, req %v", err, userId)
		}
	}

	// 更新会话记录
	if _, ok := conversations.ConversationList[conversationId]; ok {
		return nil
	}

	// 添加会话记录
	conversations.ConversationList[conversationId] = &immodels.Conversation{
		ConversationId: conversationId,
		ChatType:       chatType,
		IsShow:         isShow,
	}

	// 更新会话
	_, err = l.svcCtx.ConversationsModel.Update(l.ctx, conversations)
	if err != nil {
		return errors.Wrapf(xerr.NewDBErr(), "ConversationsModel.Update err %v, req %v", err, conversations)
	}

	return nil
}
