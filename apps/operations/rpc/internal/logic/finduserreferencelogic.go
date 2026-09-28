package logic

import (
	"context"

	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/user/rpc/user"
)

type FindUserReferenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFindUserReferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FindUserReferenceLogic {
	return &FindUserReferenceLogic{ctx: ctx, svcCtx: svcCtx}
}

// FindUserReference 排障用最小用户引用。
// 禁止返回 password / token / phone / avatar / 完整 UserEntity。
func (l *FindUserReferenceLogic) FindUserReference(in *operations.FindUserReferenceRequest) (*operations.FindUserReferenceResponse, error) {
	if in == nil {
		return nil, types.InvalidArgument("request is required")
	}
	if in.UserId == "" && in.Nickname == "" && in.Phone == "" {
		return nil, types.InvalidArgument("one of userId/nickname/phone is required")
	}

	limit, err := types.ValidateLimit(in.Limit, l.svcCtx.DefaultLimit(), l.svcCtx.MaxLimit())
	if err != nil {
		return nil, err
	}

	req := &user.FindUserReq{
		Name:  in.Nickname,
		Phone: in.Phone,
	}
	if in.UserId != "" {
		req.Ids = []string{in.UserId}
	}

	resp, err := l.svcCtx.UserRpc.FindUser(l.ctx, req)
	if err != nil {
		return nil, types.MapQueryError(err)
	}

	observedAt := nowUnixNano()
	users := make([]*operations.UserReference, 0, len(resp.Users))
	for _, u := range resp.Users {
		if int32(len(users)) >= limit {
			break
		}
		// 只保留最小字段，禁止透传 UserEntity 敏感项
		users = append(users, &operations.UserReference{
			UserId:      u.Id,
			DisplayName: u.Nickname,
			Status:      u.Status,
			ObservedAt:  observedAt,
		})
	}

	return &operations.FindUserReferenceResponse{Users: users}, nil
}
