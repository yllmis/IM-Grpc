package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPolicyRetriesWithBackoff(t *testing.T) {
	attempts := 0
	err := (Policy{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond}).Run(
		context.Background(),
		func(_ int) error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary")
			}
			return nil
		}, nil)
	if err != nil || attempts != 3 {
		t.Fatalf("retry attempts=%d err=%v", attempts, err)
	}
}
