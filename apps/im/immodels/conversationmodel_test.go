package immodels

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Explicitly uses the test DB. Run in the test container with its existing URL.
func TestConversationUpdateConcurrentIntegration(t *testing.T) {
	url := os.Getenv("LOADTEST_MONGO_URL")
	if url == "" {
		t.Skip("requires LOADTEST_MONGO_URL and MongoDB 4.2+")
	}
	logx.SetLevel(logx.ErrorLevel)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(url))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Disconnect(context.Background()) })
	collection := client.Database("yllmis-im-test").Collection("conversation")
	id := "loadtest_model_" + bson.NewObjectID().Hex()
	filter := bson.M{"conversationId": id}
	if _, err := collection.InsertOne(ctx, bson.M{"conversationId": id, "total": 0}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := collection.DeleteOne(cleanupCtx, filter); err != nil {
			t.Errorf("test fixture cleanup: %v", err)
		}
	})
	model := MustConversationModel(url, "yllmis-im-test")
	update := func(sendTime int64) error {
		return model.UpdateMsg(ctx, &ChatLog{ID: bson.NewObjectID(), ConversationId: id,
			SendTime: sendTime, MsgContent: "$literal-content", SendId: "test-sender"})
	}
	if err := update(20); err != nil {
		t.Fatal(err)
	}
	if err := update(10); err != nil {
		t.Fatal(err)
	}
	var first Conversation
	if err := collection.FindOne(ctx, filter).Decode(&first); err != nil {
		t.Fatal(err)
	}
	if first.Total != 2 || first.Msg == nil || first.Msg.SendTime != 20 {
		t.Fatal("older message overwrote the newer summary or count is incorrect")
	}
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	for i := int64(21); i <= 120; i++ {
		wg.Add(1)
		go func(value int64) {
			defer wg.Done()
			if err := update(value); err != nil {
				errors <- err
			}
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	var final Conversation
	if err := collection.FindOne(ctx, filter).Decode(&final); err != nil {
		t.Fatal(err)
	}
	if final.Total != 102 || final.Msg == nil || final.Msg.SendTime != 120 || final.Msg.MsgContent != "$literal-content" {
		t.Fatalf("atomic count/latest-summary failed: total=%d msg=%+v", final.Total, final.Msg)
	}
}

func TestChatLogInsertIfAbsentIntegration(t *testing.T) {
	url := os.Getenv("LOADTEST_MONGO_URL")
	if url == "" {
		t.Skip("requires LOADTEST_MONGO_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	id := bson.NewObjectID()
	model := MustChatLogModel(url, "yllmis-im-test")
	log := &ChatLog{ID: id, ConversationId: "idempotency_" + id.Hex(), SendId: "sender", RecvId: "receiver", SendTime: 1}
	inserted, err := model.InsertIfAbsent(ctx, log)
	if err != nil || !inserted {
		t.Fatalf("first insert inserted=%t err=%v", inserted, err)
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_, _ = model.Delete(cleanupCtx, id.Hex())
	})
	inserted, err = model.InsertIfAbsent(ctx, log)
	if err != nil || inserted {
		t.Fatalf("duplicate insert inserted=%t err=%v", inserted, err)
	}
}
