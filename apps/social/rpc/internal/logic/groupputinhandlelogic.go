package logic

import (
	"context"
	"database/sql"

	"github.com/IM_System/apps/social/rpc/internal/svc"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/social/socialmodels"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/useraccess"
	"github.com/IM_System/pkg/xerr"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	ErrGroupReqBeforePass   = xerr.NewMsg("请求已通过")
	ErrGroupReqBeforeRefuse = xerr.NewMsg("请求已拒绝")
)

type GroupPutInHandleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGroupPutInHandleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupPutInHandleLogic {
	return &GroupPutInHandleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GroupPutInHandleLogic) GroupPutInHandle(in *social.GroupPutInHandleReq) (*social.GroupPutInHandleResp, error) {
	// todo: add your logic here and delete this line

	groupReq, err := l.svcCtx.GroupRequestsModel.FindOne(l.ctx, int64(in.GroupReqId))
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "find group req err %v req %v", err, in.GroupReqId)
	}
	if in.HandleUid == "" || groupReq == nil || in.GroupId != groupReq.GroupId {
		return nil, useraccess.Denied()
	}
	member, err := l.svcCtx.GroupMembersModel.FindByGroupIdAndUserId(l.ctx, in.HandleUid, groupReq.GroupId)
	if err != nil {
		if errors.Is(err, socialmodels.ErrNotFound) {
			return nil, useraccess.Denied()
		}
		return nil, errors.Wrap(xerr.NewDBErr(), "查询审批人群权限失败")
	}
	if member == nil || member.UserId != in.HandleUid || member.GroupId != groupReq.GroupId || (constants.GroupRoleLevel(member.RoleLevel) != constants.CreatorGroupRoleLevel && constants.GroupRoleLevel(member.RoleLevel) != constants.ManagerGroupRoleLevel) {
		return nil, useraccess.Denied()
	}
	if constants.HandlerResult(in.HandleResult) != constants.PassHandlerResult && constants.HandlerResult(in.HandleResult) != constants.RefuseHandlerResult {
		return nil, status.Error(codes.InvalidArgument, "不支持的审批结果")
	}

	switch constants.HandlerResult(groupReq.HandleResult.Int64) {
	case constants.PassHandlerResult:
		return nil, errors.WithStack(ErrGroupReqBeforePass)
	case constants.RefuseHandlerResult:
		return nil, errors.WithStack(ErrGroupReqBeforeRefuse)
	}

	groupReq.HandleResult = sql.NullInt64{
		Int64: int64(in.HandleResult),
		Valid: true,
	}

	err = l.svcCtx.GroupRequestsModel.Trans(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := l.svcCtx.GroupRequestsModel.Update(l.ctx, session, groupReq); err != nil {
			return errors.Wrapf(xerr.NewDBErr(), "update group req err %v req %v", err, groupReq)
		}

		if constants.HandlerResult(groupReq.HandleResult.Int64) != constants.PassHandlerResult {
			return nil
		}

		groupMember := &socialmodels.GroupMembers{
			GroupId:     groupReq.GroupId,
			UserId:      groupReq.ReqId,
			RoleLevel:   int(constants.AtLargeGroupRoleLevel),
			OperatorUid: in.HandleUid,
		}
		_, err = l.svcCtx.GroupMembersModel.Insert(l.ctx, session, groupMember)
		if err != nil {
			return errors.Wrapf(xerr.NewDBErr(), "insert group member err %v req %v", err, groupReq)
		}
		return nil
	})

	return &social.GroupPutInHandleResp{
		GroupId: groupReq.GroupId,
		ReqId:   groupReq.ReqId,
	}, err
}
