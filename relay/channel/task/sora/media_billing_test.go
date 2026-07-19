package sora

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSeedanceTestContext(t *testing.T, body string) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	_, err := common.GetBodyStorage(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		common.CleanupBodyStorage(ctx)
	})
	return ctx, &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
}

func TestSeedanceValidationAndBillingDimensions(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-standard","prompt":"demo","duration":15,"resolution":"4K"}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, float64(1), dimensions.Units)
	assert.Equal(t, float64(15), dimensions.Seconds)
	assert.Equal(t, "4k", dimensions.ResolutionTier)
}

func TestSeedanceRejectsUnsupportedResolution(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-fast","prompt":"demo","duration":5,"resolution":"1080p"}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_resolution", taskErr.Code)
}

func TestSeedanceRejectsDurationOutsideProviderRange(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-mini","prompt":"demo","duration":16,"resolution":"720p"}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_seconds", taskErr.Code)
}

func TestSeedanceRejectsMissingDuration(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-mini","prompt":"demo","resolution":"720p"}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_seconds", taskErr.Code)
}

func TestSeedanceBuildRequestBodyUsesNormalizedParameters(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-standard","prompt":"demo","duration":"15","resolution":"4K"}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	info.UpstreamModelName = "videos-standard"

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "4k", upstream["resolution"])
	assert.Equal(t, float64(15), upstream["duration"])
}

func TestSeedanceBuildRequestBodyConvertsSecondsAliasToDuration(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-fast","prompt":"demo","seconds":"5","resolution":"720P"}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	info.UpstreamModelName = "videos-fast"

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, float64(5), upstream["duration"])
	assert.Equal(t, "720p", upstream["resolution"])
	assert.NotContains(t, upstream, "seconds")
}

func TestSeedanceCompletionIgnoresUnknownResolution(t *testing.T) {
	dimensions := (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(nil, &relaycommon.TaskInfo{
		Duration:   8,
		Resolution: "unknown-cheap-tier",
	})
	require.NotNil(t, dimensions)
	assert.Equal(t, float64(8), dimensions.Seconds)
	assert.Empty(t, dimensions.ResolutionTier)
}

func TestSeedanceCompletionRejectsResolutionUnsupportedByOriginalModel(t *testing.T) {
	task := &model.Task{Properties: model.Properties{OriginModelName: "videos-mini"}}
	dimensions := (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
		Resolution: "1080p",
	})

	assert.Nil(t, dimensions)
}
