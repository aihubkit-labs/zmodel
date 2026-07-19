package sora

import (
	"fmt"
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
	tests := []struct {
		model      string
		resolution string
	}{
		{model: "videos-mini", resolution: "480p"},
		{model: "videos-mini", resolution: "720p"},
		{model: "videos-fast", resolution: "480p"},
		{model: "videos-fast", resolution: "720p"},
		{model: "videos-standard", resolution: "480p"},
		{model: "videos-standard", resolution: "720p"},
		{model: "videos-standard", resolution: "1080p"},
		{model: "videos-standard", resolution: "4K"},
		{model: "videos-4-mini", resolution: "480p"},
		{model: "videos-4-mini", resolution: "720p"},
		{model: "videos-4-fast", resolution: "480p"},
		{model: "videos-4-fast", resolution: "720p"},
		{model: "videos-4", resolution: "480p"},
		{model: "videos-4", resolution: "720p"},
	}

	for _, test := range tests {
		t.Run(test.model+"_"+test.resolution, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, fmt.Sprintf(
				`{"model":%q,"prompt":"demo","duration":15,"resolution":%q}`,
				test.model, test.resolution,
			))
			adaptor := &TaskAdaptor{}
			require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

			dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
			require.NoError(t, err)
			assert.Equal(t, float64(1), dimensions.Units)
			assert.Equal(t, float64(15), dimensions.Seconds)
			assert.Equal(t, strings.ToLower(test.resolution), dimensions.ResolutionTier)
		})
	}
}

func TestSeedanceRejectsUnsupportedResolution(t *testing.T) {
	for _, modelName := range []string{"videos-fast", "videos-4-mini", "videos-4-fast", "videos-4"} {
		t.Run(modelName, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, fmt.Sprintf(
				`{"model":%q,"prompt":"demo","duration":5,"resolution":"1080p"}`,
				modelName,
			))
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, "invalid_resolution", taskErr.Code)
		})
	}
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

func TestSeedanceBuildRequestBodyPreservesReferenceMedia(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"videos-4",
		"prompt":"demo",
		"duration":5,
		"resolution":"720p",
		"referenceImages":["https://example.com/1.jpg","https://example.com/2.webp"],
		"referenceVideos":["https://example.com/1.mp4"],
		"referenceAudios":["https://example.com/1.mp3"]
	}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	info.UpstreamModelName = "videos-4"
	parsed, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/1.jpg", "https://example.com/2.webp"}, parsed.ReferenceImages)
	assert.Equal(t, []string{"https://example.com/1.mp4"}, parsed.ReferenceVideos)
	assert.Equal(t, []string{"https://example.com/1.mp3"}, parsed.ReferenceAudios)

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var upstream struct {
		ReferenceImages []string `json:"referenceImages"`
		ReferenceVideos []string `json:"referenceVideos"`
		ReferenceAudios []string `json:"referenceAudios"`
	}
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, []string{"https://example.com/1.jpg", "https://example.com/2.webp"}, upstream.ReferenceImages)
	assert.Equal(t, []string{"https://example.com/1.mp4"}, upstream.ReferenceVideos)
	assert.Equal(t, []string{"https://example.com/1.mp3"}, upstream.ReferenceAudios)
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
	for _, modelName := range []string{"videos-mini", "videos-fast", "videos-4-mini", "videos-4-fast", "videos-4"} {
		t.Run(modelName, func(t *testing.T) {
			task := &model.Task{Properties: model.Properties{OriginModelName: modelName}}
			dimensions := (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
				Resolution: "1080p",
			})

			assert.Nil(t, dimensions)
		})
	}
}
