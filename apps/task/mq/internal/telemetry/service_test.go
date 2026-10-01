package telemetry

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
)

type fakeKafkaClient struct{}

func (fakeKafkaClient) Metadata(context.Context, *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
	return &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "chat", Partitions: []kafka.Partition{{ID: 0}, {ID: 1}}}}}, nil
}

func (fakeKafkaClient) ListOffsets(context.Context, *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
	return &kafka.ListOffsetsResponse{Topics: map[string][]kafka.PartitionOffsets{
		"chat": {{Partition: 0, LastOffset: 120}, {Partition: 1, LastOffset: 80}},
	}}, nil
}

func (fakeKafkaClient) OffsetFetch(context.Context, *kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
	return &kafka.OffsetFetchResponse{Topics: map[string][]kafka.OffsetFetchPartition{
		"chat": {{Partition: 0, CommittedOffset: 100}, {Partition: 1, CommittedOffset: 40}},
	}}, nil
}

func TestCollectLag(t *testing.T) {
	lags, err := CollectLag(context.Background(), fakeKafkaClient{}, "chat", "group")
	if err != nil {
		t.Fatal(err)
	}
	if lags[0] != 20 || lags[1] != 40 {
		t.Fatalf("unexpected lags: %#v", lags)
	}
}
