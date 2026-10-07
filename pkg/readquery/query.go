// Package readquery contains domain-neutral bounded-query rules.
// It must not import OperationsQuery or domain implementations.
package readquery

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

const Timeout = 3 * time.Second

func MapError(err error) error {
	if err == nil {
		return nil
	}
	if s, ok := status.FromError(err); ok {
		return s.Err()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "query deadline exceeded")
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "query canceled")
	}
	msg := err.Error()
	if strings.Contains(msg, "context deadline exceeded") {
		return status.Error(codes.DeadlineExceeded, "query deadline exceeded")
	}
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "server selection error") || strings.Contains(msg, "topology is closed") {
		return status.Error(codes.Unavailable, "storage unavailable")
	}
	return status.Error(codes.Internal, "internal query error")
}
func ValidateMessageID(id string) error {
	if _, err := bson.ObjectIDFromHex(id); err != nil {
		return status.Error(codes.InvalidArgument, "messageId must be 24hex ObjectID")
	}
	return nil
}
func Limit(limit, def, max int32) (int32, error) {
	if limit < 0 || limit > max {
		return 0, status.Error(codes.InvalidArgument, "invalid query limit")
	}
	if limit == 0 {
		return def, nil
	}
	return limit, nil
}
func ValidIdentifier(s string) bool {
	return s != "" && s == strings.TrimSpace(s) && len(s) <= 128 && !strings.ContainsAny(s, "\x00\n\r")
}
