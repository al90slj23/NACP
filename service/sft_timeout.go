package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// RelayRequestStartTime returns the earliest server-side timestamp for a relay
// request. The SFT retry budget is measured from this point because the client
// is already waiting once NACP has received the request.
func RelayRequestStartTime(c *gin.Context) time.Time {
	if c == nil {
		return time.Now()
	}
	startTime := common.GetContextKeyTime(c, constant.ContextKeyRelayReceivedAt)
	if startTime.IsZero() {
		startTime = common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
	}
	if startTime.IsZero() {
		startTime = time.Now()
	}
	return startTime
}

func SFTRetryTimeoutRemaining(c *gin.Context, cfg *ChannelHealthConfig) (time.Duration, bool) {
	if cfg == nil || cfg.TotalRetryTimeout <= 0 {
		return 0, false
	}
	remaining := cfg.TotalRetryTimeout - time.Since(RelayRequestStartTime(c))
	return remaining, true
}

func SFTRetryTimedOut(c *gin.Context, cfg *ChannelHealthConfig) bool {
	remaining, enabled := SFTRetryTimeoutRemaining(c, cfg)
	return enabled && remaining <= 0
}

func SFTFirstByteTimeout(c *gin.Context, cfg *ChannelHealthConfig) (time.Duration, bool) {
	if cfg == nil {
		cfg = GetHealthConfig()
	}

	timeout := time.Duration(0)
	enabled := false
	if cfg != nil && cfg.FirstByteTimeout > 0 {
		timeout = cfg.FirstByteTimeout
		enabled = true
	}

	if remaining, totalEnabled := SFTRetryTimeoutRemaining(c, cfg); totalEnabled {
		if !enabled || remaining < timeout {
			timeout = remaining
		}
		enabled = true
	}

	return timeout, enabled
}

type FirstByteTimeoutGuard struct {
	cancel   context.CancelFunc
	timeout  time.Duration
	timer    *time.Timer
	timedOut atomic.Bool
	once     sync.Once
}

func NewSFTFirstByteTimeoutContext(c *gin.Context, parent context.Context, cfg *ChannelHealthConfig) (context.Context, *FirstByteTimeoutGuard) {
	timeout, enabled := SFTFirstByteTimeout(c, cfg)
	if !enabled {
		return parent, nil
	}

	ctx, cancel := context.WithCancel(parent)
	guard := &FirstByteTimeoutGuard{
		cancel:  cancel,
		timeout: timeout,
	}
	if timeout <= 0 {
		guard.timedOut.Store(true)
		cancel()
		return ctx, guard
	}

	guard.timer = time.AfterFunc(timeout, func() {
		guard.timedOut.Store(true)
		cancel()
	})
	return ctx, guard
}

func (g *FirstByteTimeoutGuard) StopFirstByteTimer() {
	if g == nil || g.timer == nil {
		return
	}
	g.timer.Stop()
}

func (g *FirstByteTimeoutGuard) Cancel() {
	if g == nil {
		return
	}
	g.once.Do(g.cancel)
}

func (g *FirstByteTimeoutGuard) TimedOut() bool {
	return g != nil && g.timedOut.Load()
}

func (g *FirstByteTimeoutGuard) Timeout() time.Duration {
	if g == nil {
		return 0
	}
	return g.timeout
}

func (g *FirstByteTimeoutGuard) WrapResponseBody(resp *http.Response) {
	if g == nil || resp == nil || resp.Body == nil {
		return
	}
	resp.Body = &cancelOnCloseReadCloser{
		ReadCloser: resp.Body,
		cancel:     g.Cancel,
	}
}

type cancelOnCloseReadCloser struct {
	io.ReadCloser
	cancel func()
	once   sync.Once
}

func (r *cancelOnCloseReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.once.Do(r.cancel)
	return err
}

func NewSFTTotalTimeoutError(c *gin.Context, cfg *ChannelHealthConfig) *types.NewAPIError {
	remaining, enabled := SFTRetryTimeoutRemaining(c, cfg)
	elapsed := time.Since(RelayRequestStartTime(c))
	if enabled && remaining > 0 {
		elapsed = cfg.TotalRetryTimeout - remaining
	}
	return types.NewErrorWithStatusCode(
		fmt.Errorf("SFT retry total timeout exceeded after %.2f seconds", elapsed.Seconds()),
		types.ErrorCodeChannelResponseTimeExceeded,
		http.StatusGatewayTimeout,
		types.ErrOptionWithHideErrMsg("upstream timeout: retry queue total timeout"),
	)
}

func NewSFTFirstByteTimeoutError(timeout time.Duration) *types.NewAPIError {
	if timeout < 0 {
		timeout = 0
	}
	return types.NewErrorWithStatusCode(
		fmt.Errorf("upstream first byte timeout after %.2f seconds", timeout.Seconds()),
		types.ErrorCodeChannelResponseTimeExceeded,
		http.StatusGatewayTimeout,
		types.ErrOptionWithHideErrMsg("upstream timeout: first byte timeout"),
	)
}
