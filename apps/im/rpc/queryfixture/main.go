// queryfixture 是隔离联调进程：使用真实 Model/Handler，但不注册 etcd，
// 不启动业务服务，不调用 Kafka/推送，也不允许连接正常聊天数据库。
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/im/rpc/internal/server"
	"github.com/IM_System/apps/im/rpc/internal/svc"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/serviceauth"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/grpc"
)

func run() error {
	db, token := os.Getenv("QUERY_TEST_DB"), os.Getenv("QUERY_TEST_TOKEN")
	if !strings.HasPrefix(db, "query_contract_test_") || token == "" {
		return fmt.Errorf("isolated test database and token required")
	}
	url := os.Getenv("QUERY_TEST_MONGO_URL")
	client, err := mongo.Connect(options.Client().ApplyURI(url))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer client.Disconnect(context.Background())
	coll := client.Database(db).Collection("chat_log")
	// 六条数据覆盖闭区间边界、同时间稳定排序和 limit+1 截断。
	const base int64 = 1791417600000000000
	for i, delta := range []int64{-1, 0, 50, 50, 100, 101} {
		id, _ := bson.ObjectIDFromHex(fmt.Sprintf("%024x", i+1))
		row := &immodels.ChatLog{ID: id, ConversationId: "query-fixture-conversation", SendId: "query-fixture-u1", RecvId: "query-fixture-u2", SendTime: base + delta, ChatType: constants.SingleChatType, MsgContent: "fixture-body-must-not-leak", ReadRecords: []byte{1}}
		if _, err := coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$setOnInsert": row}, options.UpdateOne().SetUpsert(true)); err != nil {
			return err
		}
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(serviceauth.Guard(serviceauth.Config{Enable: true, Token: token}, "im.MessageQuery")))
	im.RegisterMessageQueryServer(s, server.NewMessageQueryServer(&svc.ServiceContext{ChatLogModel: immodels.MustChatLogModel(url, db), MessageSearch: immodels.MustMessageSearchModel(url, db)}))
	lis, err := net.Listen("tcp", ":9100")
	if err != nil {
		return err
	}
	fmt.Println("isolated_message_query_ready=true")
	return s.Serve(lis)
}

func main() {
	if err := run(); err != nil {
		// 数据源错误可能携带凭证，不输出原始错误。
		fmt.Println("isolated_message_query_failed=true")
		os.Exit(1)
	}
}
