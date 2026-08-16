package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_SuccessFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Config{Attempts: 3}, "test", func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_RetriesThenSuccess(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Config{Attempts: 5, InitialWait: time.Millisecond, Force: true}, "test", func() error {
		calls++
		if calls < 3 {
			return errors.New("not ready")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_ExhaustsAttempts(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Config{Attempts: 3, InitialWait: time.Millisecond, Force: true}, "test", func() error {
		calls++
		return errors.New("boom")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sau 3 lần thử")
	assert.Contains(t, err.Error(), "boom")
	assert.Equal(t, 3, calls)
}

func TestDo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // đã cancelled từ đầu
	err := Do(ctx, Config{Attempts: 5, InitialWait: time.Millisecond, Force: true}, "test", func() error {
		return errors.New("boom")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dừng retry vì context cancelled")
}

func TestDo_DefaultsApplied(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Config{Force: true}, "test", func() error {
		calls++
		if calls < 10 {
			return errors.New("x")
		}
		return nil
	})
	require.NoError(t, err) // mặc định 10 attempts — lần 10 thành công
	assert.Equal(t, 10, calls)
}

func TestDo_SkipRetryInCI(t *testing.T) {
	t.Setenv("CI", "true")
	calls := 0
	err := Do(context.Background(), Config{Attempts: 5, InitialWait: time.Millisecond}, "test", func() error {
		calls++
		return errors.New("boom")
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls, "CI → báo lỗi luôn, không retry")
	assert.Contains(t, err.Error(), "sau 1 lần thử")
}

func TestSkipRetry_Detect(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("APP_ENV", "testing")
	assert.True(t, SkipRetry(), "APP_ENV=testing → skip retry")

	t.Setenv("APP_ENV", "development")
	assert.False(t, SkipRetry(), "development → retry bình thường")
}
