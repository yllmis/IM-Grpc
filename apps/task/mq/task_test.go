package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IM_System/apps/task/mq/internal/config"
)

func localTestConfigPath(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("etc/loadtest/task.yaml.example")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "task.yaml")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLocalConfigLoads(t *testing.T) {
	t.Setenv("LOADTEST_MONGO_URL", "mongodb://localhost:27017")
	t.Setenv("LOADTEST_REDIS_PASSWORD", "test-only")
	c, err := loadTaskConfig(localTestConfigPath(t), "local", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Mongo.Db != "yllmis-im-test" || c.MsgChatTransfer.Processors != 8 || !c.LoadTest.PersistenceOnly {
		t.Fatalf("isolated test routing was not loaded: %+v", c)
	}
	if _, err := loadTaskConfig("unused.yaml", "invalid", nil); err == nil {
		t.Fatal("invalid config source accepted")
	}
	if _, err := loadTaskConfig("missing.yaml", "local", nil); err == nil {
		t.Fatal("missing local file accepted")
	}
}

func TestConfigValidation(t *testing.T) {
	c := config.Config{}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.MessageProcessingRateLimit = config.MessageProcessingRateLimitConfig{
		Enabled: true, GlobalMessagesPerSecond: 600, ExpectedInstances: 2, Burst: 1,
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := c.PerInstanceMessagesPerSecond(); got != 300 {
		t.Fatalf("per-instance rate=%v, want 300", got)
	}
	for _, bad := range []config.Config{
		{MessageProcessingRateLimit: config.MessageProcessingRateLimitConfig{Enabled: true, Burst: 1}},
		{MessageProcessingRateLimit: config.MessageProcessingRateLimitConfig{Enabled: true, GlobalMessagesPerSecond: 1, ExpectedInstances: 0, Burst: 1}},
		{MessageRetry: config.MessageRetryConfig{Enabled: true, MaxAttempts: 0, InitialBackoffMs: 1, MaxBackoffMs: 2, DeadLetterTopic: "dlq"}},
		{MessageRetry: config.MessageRetryConfig{Enabled: true, MaxAttempts: 1, InitialBackoffMs: 20, MaxBackoffMs: 10, DeadLetterTopic: "dlq"}},
		{MessageRetry: config.MessageRetryConfig{Enabled: true, MaxAttempts: 1, InitialBackoffMs: 1, MaxBackoffMs: 2}},
	} {
		if err := bad.Validate(); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
}
