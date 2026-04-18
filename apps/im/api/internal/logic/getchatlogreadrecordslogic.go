// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/IM_System/apps/im/api/internal/svc"
	"github.com/IM_System/apps/im/api/internal/types"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/bitmap"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetChatLogReadRecordsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetChatLogReadRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChatLogReadRecordsLogic {
	return &GetChatLogReadRecordsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetChatLogReadRecordsLogic) GetChatLogReadRecords(req *types.GetChatLogReadRecordsReq) (resp *types.GetChatLogReadRecordsResp, err error) {
	// todo: add your logic here and delete this line

	chatlogs, err := l.svcCtx.Im.GetChatLog(l.ctx, &im.GetChatLogReq{
		MsgId: req.MsgId,
	})
	if err != nil {
		return nil, err
	}
	if len(chatlogs.List) == 0 {
		return nil, xerr.NewMsg("消息不存在")
	}

	var (
		chatlog = chatlogs.List[0]
		reads   = []string{chatlog.SendId}
		unreads []string
		ids     []string
	)

	switch constants.ChatType(chatlog.ChatType) {
	case constants.SingleChatType:
		// 私聊: 发送者已读, 接收者未读
		if len(chatlog.ReadRecords) == 0 || chatlog.ReadRecords[0] == 0 {
			unreads = []string{chatlog.RecvId}
		} else {
			reads = append(reads, chatlog.RecvId)
		}
		ids = []string{chatlog.SendId, chatlog.RecvId}
	case constants.GroupChatType:
		// 群聊: 发送者已读, readRecords中用户Id对应的值为1则已读, 0则未读
		groupUser, err := l.svcCtx.Social.Groupusers(l.ctx, &social.GroupusersReq{
			GroupId: chatlog.RecvId,
		})
		if err != nil {
			return nil, err
		}

		bitmaps := bitmap.Load(chatlog.ReadRecords)
		for _, members := range groupUser.List {
			ids = append(ids, members.UserId)

			if members.UserId == chatlog.SendId {
				continue
			}

			if bitmaps.IsSet(members.UserId) {
				reads = append(reads, members.UserId)
			} else {
				unreads = append(unreads, members.UserId)
			}
		}
	}

	userEntity, err := l.svcCtx.User.FindUser(l.ctx, &user.FindUserReq{
		Ids: ids,
	})
	if err != nil {
		return nil, err
	}
	userEntitySet := make(map[string]*user.UserEntity, len(userEntity.Users))
	for i, entity := range userEntity.Users {
		userEntitySet[entity.Id] = userEntity.Users[i]
	}

	resp = &types.GetChatLogReadRecordsResp{
		Reads:   reads,
		UnReads: unreads,
	}
	return resp, nil
}
