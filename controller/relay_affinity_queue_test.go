package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldExcludeSequentialAttemptKeepsAffinityPrependOutOfBaseQueue(t *testing.T) {
	const affinityChannelID = 10
	pending := true

	require.False(t, shouldExcludeSequentialAttempt(affinityChannelID, affinityChannelID, &pending))
	require.False(t, pending)

	require.True(t, shouldExcludeSequentialAttempt(30, affinityChannelID, &pending))
	require.True(t, shouldExcludeSequentialAttempt(40, affinityChannelID, &pending))
	require.True(t, shouldExcludeSequentialAttempt(affinityChannelID, affinityChannelID, &pending))
	require.True(t, shouldExcludeSequentialAttempt(20, affinityChannelID, &pending))
}

func TestShouldExcludeSequentialAttemptWithoutAffinityExcludesNormally(t *testing.T) {
	pending := false

	require.True(t, shouldExcludeSequentialAttempt(30, 0, &pending))
	require.True(t, shouldExcludeSequentialAttempt(40, 0, &pending))
	require.True(t, shouldExcludeSequentialAttempt(10, 0, &pending))
	require.True(t, shouldExcludeSequentialAttempt(20, 0, &pending))
}

func TestGetChannelFirstAttemptUsesAlreadySelectedChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("channel_id", 88)
	ctx.Set("channel_type", 1)
	ctx.Set("channel_name", "selected-before-relay")

	channel, err := getChannel(ctx, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:   88,
			ChannelType: 1,
		},
	}, &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "default",
		ModelName:  "gpt-a",
		Retry:      common.GetPointer(0),
		ExcludeIDs: map[int]bool{},
	})

	require.Nil(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 88, channel.Id)
	require.Equal(t, "selected-before-relay", channel.Name)
}
