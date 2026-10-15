package logic

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/IM_System/apps/social/rpc/internal/svc"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/social/socialmodels"
	"github.com/IM_System/pkg/constants"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type friendApprovalModel struct {
	socialmodels.FriendRequestsModel
	trans, updates int
	transErr       error
}

func (*friendApprovalModel) FindOne(context.Context, int64) (*socialmodels.FriendRequests, error) {
	return &socialmodels.FriendRequests{UserId: "sender", ReqUid: "recipient", HandleResult: sql.NullInt64{Int64: int64(constants.NoHandlerResult), Valid: true}}, nil
}
func (f *friendApprovalModel) Trans(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	f.trans++
	if f.transErr != nil {
		return f.transErr
	}
	return fn(ctx, nil)
}
func (f *friendApprovalModel) Update(context.Context, sqlx.Session, *socialmodels.FriendRequests) error {
	f.updates++
	return nil
}

func TestFriendApprovalOnlyRecipient(t *testing.T) {
	for _, uid := range []string{"", "sender", "outsider", "recipient"} {
		f := &friendApprovalModel{}
		_, err := NewFriendPutInHandleLogic(context.Background(), &svc.ServiceContext{FriendRequestsModel: f}).FriendPutInHandle(&social.FriendPutInHandleReq{UserId: uid, FriendReqId: 1, HandleResult: int32(constants.RefuseHandlerResult)})
		if uid == "recipient" {
			if err != nil || f.updates != 1 {
				t.Fatal(err)
			}
		} else if status.Code(err) != codes.PermissionDenied || f.trans != 0 {
			t.Fatal("unauthorized approval reached transaction")
		}
	}
	f := &friendApprovalModel{transErr: errors.New("transaction failed")}
	if _, err := NewFriendPutInHandleLogic(context.Background(), &svc.ServiceContext{FriendRequestsModel: f}).FriendPutInHandle(&social.FriendPutInHandleReq{UserId: "recipient", FriendReqId: 1, HandleResult: int32(constants.RefuseHandlerResult)}); err == nil {
		t.Fatal("transaction failure became success")
	}
}

type groupApprovalModel struct {
	socialmodels.GroupRequestsModel
	trans, updates int
}

func (*groupApprovalModel) FindOne(context.Context, int64) (*socialmodels.GroupRequests, error) {
	return &socialmodels.GroupRequests{GroupId: "g1", ReqId: "applicant", HandleResult: sql.NullInt64{Int64: int64(constants.NoHandlerResult), Valid: true}}, nil
}
func (f *groupApprovalModel) Trans(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	f.trans++
	return fn(ctx, nil)
}
func (f *groupApprovalModel) Update(context.Context, sqlx.Session, *socialmodels.GroupRequests) error {
	f.updates++
	return nil
}

type approvalMemberModel struct {
	socialmodels.GroupMembersModel
	role constants.GroupRoleLevel
	err  error
}

func (f *approvalMemberModel) FindByGroupIdAndUserId(_ context.Context, uid, group string) (*socialmodels.GroupMembers, error) {
	return &socialmodels.GroupMembers{UserId: uid, GroupId: group, RoleLevel: int(f.role)}, f.err
}

func TestGroupApprovalRequiresMatchingGroupAndManager(t *testing.T) {
	for _, tc := range []struct {
		name, group string
		role        constants.GroupRoleLevel
		storageErr  error
		allowed     bool
	}{
		{"owner", "g1", constants.CreatorGroupRoleLevel, nil, true},
		{"manager", "g1", constants.ManagerGroupRoleLevel, nil, true},
		{"member", "g1", constants.AtLargeGroupRoleLevel, nil, false},
		{"wrong group", "g2", constants.CreatorGroupRoleLevel, nil, false},
		{"nonmember", "g1", constants.CreatorGroupRoleLevel, socialmodels.ErrNotFound, false},
		{"storage failure", "g1", constants.CreatorGroupRoleLevel, errors.New("offline"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &groupApprovalModel{}
			members := &approvalMemberModel{role: tc.role, err: tc.storageErr}
			_, err := NewGroupPutInHandleLogic(context.Background(), &svc.ServiceContext{GroupRequestsModel: f, GroupMembersModel: members}).GroupPutInHandle(&social.GroupPutInHandleReq{GroupReqId: 1, GroupId: tc.group, HandleUid: "actor", HandleResult: int32(constants.RefuseHandlerResult)})
			if tc.allowed {
				if err != nil || f.updates != 1 {
					t.Fatal(err)
				}
			} else if err == nil || f.trans != 0 {
				t.Fatal("unauthorized group approval reached transaction")
			}
		})
	}
}
