package logic

import (
	"context"

	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/serviceauth"
)

type FindUserReferenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFindUserReferenceLogic(ctx context.Context, s *svc.ServiceContext) *FindUserReferenceLogic {
	return &FindUserReferenceLogic{ctx: ctx, svcCtx: s}
}

// Compatibility facade: never fetch or relay a full UserEntity.
func (l *FindUserReferenceLogic) FindUserReference(in *operations.FindUserReferenceRequest) (*operations.FindUserReferenceResponse, error) {
	if in == nil {
		return nil, types.InvalidArgument("request is required")
	}
	if l.svcCtx.UserQueryRpc == nil {
		return nil, types.Unimplemented("user domain query unavailable")
	}
	// Preserve the old selector precedence during migration. The new domain
	// contract deliberately requires exactly one selector.
	req := &user.UserReferenceRequest{Limit: in.Limit}
	switch {
	case in.Phone != "":
		req.Phone = in.Phone
	case in.Nickname != "":
		req.Nickname = in.Nickname
	default:
		req.UserId = in.UserId
	}
	limit, err := types.ValidateLimit(req.Limit, l.svcCtx.DefaultLimit(), l.svcCtx.MaxLimit())
	if err != nil {
		return nil, err
	}
	req.Limit = limit
	resp, err := l.svcCtx.UserQueryRpc.FindUserReference(serviceauth.Outgoing(l.ctx, l.svcCtx.Config.DomainQueryToken), req)
	if err != nil {
		return nil, types.MapQueryError(err)
	}
	if resp == nil {
		return nil, types.Internal("empty user domain response")
	}
	result := &operations.FindUserReferenceResponse{Users: make([]*operations.UserReference, 0, len(resp.Users)), Truncated: resp.Truncated}
	for _, row := range resp.Users {
		if row == nil {
			return nil, types.Internal("invalid user domain response")
		}
		result.Users = append(result.Users, &operations.UserReference{UserId: row.UserId, DisplayName: row.DisplayName, Status: row.Status, ObservedAt: row.ObservedAt})
	}
	return result, nil
}
