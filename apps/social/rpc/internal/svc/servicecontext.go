package svc

import (
	"github.com/IM_System/apps/social/rpc/internal/config"
	"github.com/IM_System/apps/social/socialmodels"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	socialmodels.FriendsModel
	socialmodels.FriendRequestsModel
	socialmodels.GroupsModel
	socialmodels.GroupRequestsModel
	socialmodels.GroupMembersModel
}

func NewServiceContext(c config.Config) *ServiceContext {

	sqlconn := sqlx.NewMysql(c.Mysql.DataSource)

	return &ServiceContext{
		Config: c,

		FriendsModel:        socialmodels.NewFriendsModel(sqlconn, c.Cache),
		FriendRequestsModel: socialmodels.NewFriendRequestsModel(sqlconn, c.Cache),
		GroupsModel:         socialmodels.NewGroupsModel(sqlconn, c.Cache),
		GroupRequestsModel:  socialmodels.NewGroupRequestsModel(sqlconn, c.Cache),
		GroupMembersModel:   socialmodels.NewGroupMembersModel(sqlconn, c.Cache),
	}

}
