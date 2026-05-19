package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSFTFirstByteTimeoutUsesSmallerTotalRemainingBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyRelayReceivedAt, time.Now().Add(-55*time.Second))

	timeout, enabled := SFTFirstByteTimeout(ctx, &ChannelHealthConfig{
		FirstByteTimeout:  20 * time.Second,
		TotalRetryTimeout: 60 * time.Second,
	})

	require.True(t, enabled)
	require.LessOrEqual(t, timeout, 5*time.Second)
	require.Greater(t, timeout, 0*time.Second)
}

func TestSFTFirstByteTimeoutContextStopsAfterFirstByte(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyRelayReceivedAt, time.Now())

	reqCtx, guard := NewSFTFirstByteTimeoutContext(ctx, context.Background(), &ChannelHealthConfig{
		FirstByteTimeout:  20 * time.Millisecond,
		TotalRetryTimeout: time.Minute,
	})
	require.NotNil(t, guard)

	guard.StopFirstByteTimer()
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, reqCtx.Err())
	require.False(t, guard.TimedOut())
	guard.Cancel()
}

func TestSFTFirstByteTimeoutContextCancelsWhenFirstByteIsLate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyRelayReceivedAt, time.Now())

	reqCtx, guard := NewSFTFirstByteTimeoutContext(ctx, context.Background(), &ChannelHealthConfig{
		FirstByteTimeout:  20 * time.Millisecond,
		TotalRetryTimeout: time.Minute,
	})
	require.NotNil(t, guard)

	time.Sleep(50 * time.Millisecond)
	require.ErrorIs(t, reqCtx.Err(), context.Canceled)
	require.True(t, guard.TimedOut())
	guard.Cancel()
}
