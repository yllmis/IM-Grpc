package server

import (
	"context"
	"time"

	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/readquery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserQueryServer struct {
	user.UnimplementedUserQueryServer
	svcCtx *svc.ServiceContext
}

func NewUserQueryServer(s *svc.ServiceContext) *UserQueryServer { return &UserQueryServer{svcCtx: s} }
func (s *UserQueryServer) FindUserReference(ctx context.Context, in *user.UserReferenceRequest) (*user.UserReferenceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	selectors := 0
	for _, v := range []string{in.UserId, in.Nickname, in.Phone} {
		if v != "" {
			if !readquery.ValidIdentifier(v) {
				return nil, status.Error(codes.InvalidArgument, "invalid user selector")
			}
			selectors++
		}
	}
	if selectors != 1 {
		return nil, status.Error(codes.InvalidArgument, "exactly one userId/nickname/phone is required")
	}
	limit, err := readquery.Limit(in.Limit, 50, 200)
	if err != nil {
		return nil, err
	}
	if s.svcCtx.UsersModel == nil {
		return nil, status.Error(codes.Unavailable, "user query unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, readquery.Timeout)
	defer cancel()
	rows, err := s.svcCtx.UsersModel.SearchReferences(ctx, models.UserReferenceFilter{UserID: in.UserId, Nickname: in.Nickname, Phone: in.Phone, Limit: limit})
	if err != nil {
		return nil, readquery.MapError(err)
	}
	truncated := len(rows) > int(limit)
	if truncated {
		rows = rows[:limit]
	}
	observedAt := time.Now().UnixNano()
	resp := &user.UserReferenceResponse{Users: make([]*user.UserQueryReference, 0, len(rows)), Truncated: truncated}
	for _, row := range rows {
		resp.Users = append(resp.Users, &user.UserQueryReference{UserId: row.ID, DisplayName: row.DisplayName, Status: row.Status, ObservedAt: observedAt})
	}
	return resp, nil
}
