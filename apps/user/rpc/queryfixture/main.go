// 隔离用户查询联调进程：只暴露 UserQuery，不运行登录、Redis 写入或服务注册。
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/server"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/serviceauth"
	"github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc"
)

func run() error {
	dsn, token := os.Getenv("QUERY_TEST_MYSQL_DSN"), os.Getenv("QUERY_TEST_TOKEN")
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(cfg.DBName, "query_contract_test_") || token == "" {
		return fmt.Errorf("isolated test database and token required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS users (id varchar(24) PRIMARY KEY, nickname varchar(24) NOT NULL, phone varchar(20) NOT NULL, password varchar(191), avatar varchar(191), status tinyint) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	if err != nil {
		return err
	}
	for i, nickname := range []string{"fixture_%=nick", "fixture_%=nick", "fixtureAAAnick"} {
		_, err = db.ExecContext(ctx, "INSERT IGNORE INTO users (id,nickname,phone,password,avatar,status) VALUES (?,?,?,?,?,?)", fmt.Sprintf("query-fixture-u%d", i+1), nickname, fmt.Sprintf("fixture-phone-%d", i+1), "fixture-password-must-not-leak", "fixture-avatar", 1)
		if err != nil {
			return err
		}
	}
	// SearchReferences 使用 QueryRowsNoCache，不访问 Redis；此配置只满足
	// 既有 UsersModel 的构造契约，测试不调用任何带缓存的业务方法。
	model := models.NewUsersModel(sqlx.NewMysql(dsn), cache.CacheConf{{RedisConf: redis.RedisConf{Host: "127.0.0.1:6379", Type: "node", NonBlock: true}, Weight: 100}})
	s := grpc.NewServer(grpc.UnaryInterceptor(serviceauth.Guard(serviceauth.Config{Enable: true, Token: token}, "user.UserQuery")))
	user.RegisterUserQueryServer(s, server.NewUserQueryServer(&svc.ServiceContext{UsersModel: model}))
	lis, err := net.Listen("tcp", ":9100")
	if err != nil {
		return err
	}
	fmt.Println("isolated_user_query_ready=true")
	return s.Serve(lis)
}

func main() {
	if err := run(); err != nil {
		fmt.Println("isolated_user_query_failed=true")
		os.Exit(1)
	}
}
