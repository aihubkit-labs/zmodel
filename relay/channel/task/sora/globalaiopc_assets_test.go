package sora

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalAIOpcAssetPreparationPlanner(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"sd_2.5_special_v1",
		"prompt":"demo",
		"duration":4,
		"resolution":"720p",
		"referenceImages":["https://media.example.com/subject.png"],
		"referenceVideos":["https://media.example.com/motion.mp4"],
		"referenceAudios":["https://media.example.com/voice.mp3"]
	}`)
	info.UpstreamModelName = "sd_2.5_special_v1"
	configureTestVideoModel(info, dto.VideoProtocolGlobalAIOpc, info.UpstreamModelName, []string{"720p", "1080p"}, 30, 10, 10, 4, 30)
	capability := info.ChannelSetting.VideoModelCapabilities[info.UpstreamModelName]
	capability.AssetPreparationMode = dto.VideoAssetPreparationGlobalAIOpcSeedance
	capability.AudioReferenceRequiresVisualReference = common.GetPointer(false)
	info.ChannelSetting.VideoModelCapabilities[info.UpstreamModelName] = capability
	info.ChannelSetting.GlobalAIOpcAssetPreparation = &dto.GlobalAIOpcAssetPreparationSettings{TimeoutSeconds: 120}

	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info))
	requestReader, err := (&TaskAdaptor{}).BuildRequestBody(ctx, info)
	require.NoError(t, err)
	requestBody, err := io.ReadAll(requestReader)
	require.NoError(t, err)
	startedAt := common.GetTimestamp()
	preparation, err := (&TaskAdaptor{}).BuildTaskPreparation(ctx, info, requestBody)
	require.NoError(t, err)
	require.NotNil(t, preparation)
	require.Len(t, preparation.Assets, 3)
	assert.Equal(t, model.VideoTaskAsset{Field: "reference_images", Index: 0, AssetType: "Image", SourceURL: "https://media.example.com/subject.png"}, preparation.Assets[0])
	assert.Equal(t, model.VideoTaskAsset{Field: "reference_videos", Index: 0, AssetType: "Video", SourceURL: "https://media.example.com/motion.mp4"}, preparation.Assets[1])
	assert.Equal(t, model.VideoTaskAsset{Field: "reference_audios", Index: 0, AssetType: "Audio", SourceURL: "https://media.example.com/voice.mp3"}, preparation.Assets[2])
	assert.GreaterOrEqual(t, preparation.DeadlineAt, startedAt+120)
	assert.LessOrEqual(t, preparation.DeadlineAt, common.GetTimestamp()+120)

	ctx.Set("task_request", relaycommon.TaskSubmitReq{Model: info.UpstreamModelName})
	preparation, err = (&TaskAdaptor{}).BuildTaskPreparation(ctx, info, []byte(`{"model":"sd_2.5_special_v1"}`))
	require.NoError(t, err)
	assert.Nil(t, preparation)
}

func TestGlobalAIOpcAssetModelRejectsDirectAssetIDs(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"sd_2.5_discount_v1",
		"prompt":"demo",
		"duration":4,
		"resolution":"480p",
		"referenceImages":["assetId://upstream-private-asset"]
	}`)
	info.UpstreamModelName = "sd_2.5_discount_v1"
	configureTestVideoModel(info, dto.VideoProtocolGlobalAIOpc, info.UpstreamModelName, []string{"480p", "720p"}, 30, 10, 10, 4, 30)
	capability := info.ChannelSetting.VideoModelCapabilities[info.UpstreamModelName]
	capability.AssetPreparationMode = dto.VideoAssetPreparationGlobalAIOpcSeedance
	info.ChannelSetting.VideoModelCapabilities[info.UpstreamModelName] = capability

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_reference_images", taskErr.Code)
	assert.Contains(t, taskErr.Message, "HTTP or HTTPS")
}

func TestRewriteGlobalAIOpcAssetReferencesPreservesOrder(t *testing.T) {
	preparation := &model.VideoTaskPreparation{
		RequestBody: `{
			"model":"sd_2.5_special_v1",
			"reference_images":["https://example.com/a.png","https://example.com/b.png"],
			"reference_videos":["https://example.com/c.mp4"]
		}`,
		Assets: []model.VideoTaskAsset{
			{Field: "reference_images", Index: 1, AssetID: "image-b"},
			{Field: "reference_videos", Index: 0, AssetID: "video-c"},
			{Field: "reference_images", Index: 0, AssetID: "image-a"},
		},
	}

	body, err := rewriteGlobalAIOpcAssetReferences(preparation)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model":"sd_2.5_special_v1",
		"reference_images":["assetId://image-a","assetId://image-b"],
		"reference_videos":["assetId://video-c"]
	}`, string(body))
}

func TestPrepareGlobalAIOpcTaskUploadsPollsAndSubmitsWithPinnedKey(t *testing.T) {
	service.InitHttpClient()
	var requestPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requestPaths = append(requestPaths, request.URL.Path)
		assert.Equal(t, "Bearer pinned-key", request.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/kyyReactApiServer/asset/seedance2/assetUpload":
			var payload struct {
				AssetType string `json:"assetType"`
				URL       string `json:"url"`
			}
			require.NoError(t, common.DecodeJson(request.Body, &payload))
			assert.Equal(t, "Image", payload.AssetType)
			assert.Equal(t, "https://media.example.com/subject.png", payload.URL)
			_, _ = io.WriteString(w, `{"code":0,"data":{"assetId":"asset-image","assetType":"Image","materialStatus":"PROCESSING","status":"NONE"},"msg":null}`)
		case "/kyyReactApiServer/asset/seedance2/assetDetail":
			var payload struct {
				AssetID string `json:"assetId"`
			}
			require.NoError(t, common.DecodeJson(request.Body, &payload))
			assert.Equal(t, "asset-image", payload.AssetID)
			_, _ = io.WriteString(w, `{"code":0,"data":{"assetId":"asset-image","assetType":"Image","materialStatus":"READY","status":"ACTIVE"},"msg":null}`)
		case "/kyyReactApiServer/v2/model-center/tasks":
			body, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{
				"model":"sd_2.5_special_v1",
				"prompt":"demo",
				"reference_images":["assetId://asset-image"]
			}`, string(body))
			_, _ = io.WriteString(w, `{"id":"upstream-task","status":"queued","url":"https://private.example.com/result.mp4"}`)
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	channel := &model.Channel{BaseURL: common.GetPointer(server.URL)}
	task := &model.Task{PrivateData: model.TaskPrivateData{
		Key: "pinned-key",
		VideoPreparation: &model.VideoTaskPreparation{
			RequestBody: `{"model":"sd_2.5_special_v1","prompt":"demo","reference_images":["https://media.example.com/subject.png"]}`,
			DeadlineAt:  common.GetTimestamp() + 600,
			Assets: []model.VideoTaskAsset{{
				Field:     "reference_images",
				Index:     0,
				AssetType: "Image",
				SourceURL: "https://media.example.com/subject.png",
			}},
		},
	}}

	result, err := (&TaskAdaptor{}).PrepareTask(context.Background(), channel, task)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Waiting)
	assert.Equal(t, "asset-image", task.PrivateData.VideoPreparation.Assets[0].AssetID)
	assert.Equal(t, "NONE", task.PrivateData.VideoPreparation.Assets[0].Status)

	task.PrivateData.VideoPreparation.Assets[0].NextPollAt = 0
	result, err = (&TaskAdaptor{}).PrepareTask(context.Background(), channel, task)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Waiting)
	assert.Equal(t, "upstream-task", result.UpstreamTaskID)
	assert.NotContains(t, string(result.TaskData), "private.example.com")
	require.NotNil(t, result.UpstreamHTTPTrace)
	require.NotNil(t, result.UpstreamHTTPTrace.PreparationRequest)
	assert.Contains(t, result.UpstreamHTTPTrace.PreparationRequest.URL, "/asset/seedance2/assetDetail")
	require.NotNil(t, result.UpstreamHTTPTrace.SubmitRequest)
	assert.Contains(t, result.UpstreamHTTPTrace.SubmitRequest.URL, "/v2/model-center/tasks")
	assert.JSONEq(t, `{
			"model":"sd_2.5_special_v1",
			"prompt":"demo",
			"reference_images":["assetId://asset-image"]
		}`, result.UpstreamHTTPTrace.SubmitRequest.Body)
	assert.Equal(t, []string{
		"/kyyReactApiServer/asset/seedance2/assetUpload",
		"/kyyReactApiServer/asset/seedance2/assetDetail",
		"/kyyReactApiServer/v2/model-center/tasks",
	}, requestPaths)
}

func TestPrepareGlobalAIOpcTaskStopsOnFailedAsset(t *testing.T) {
	service.InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/kyyReactApiServer/asset/seedance2/assetUpload", request.URL.Path)
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"assetId":"asset-failed","materialStatus":"FAILED","status":"FAILED","errorMessage":"unsupported media"},"msg":null}`)
	}))
	t.Cleanup(server.Close)

	task := &model.Task{PrivateData: model.TaskPrivateData{
		Key: "pinned-key",
		VideoPreparation: &model.VideoTaskPreparation{
			RequestBody: `{"model":"sd_2.5_discount_v1","reference_images":["https://media.example.com/bad.png"]}`,
			DeadlineAt:  common.GetTimestamp() + 600,
			Assets: []model.VideoTaskAsset{{
				Field:     "reference_images",
				Index:     0,
				AssetType: "Image",
				SourceURL: "https://media.example.com/bad.png",
			}},
		},
	}}

	_, err := (&TaskAdaptor{}).PrepareTask(context.Background(), &model.Channel{BaseURL: common.GetPointer(server.URL)}, task)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "FAILED"))
	var preparationErr *service.TaskPreparationError
	require.ErrorAs(t, err, &preparationErr)
	assert.Equal(t, "unsupported media", preparationErr.PublicMessage)
	require.NotNil(t, task.PrivateData.UpstreamHTTPTrace)
	require.NotNil(t, task.PrivateData.UpstreamHTTPTrace.PreparationRequest)
	assert.Equal(t, http.MethodPost, task.PrivateData.UpstreamHTTPTrace.PreparationRequest.Method)
	assert.Contains(t, task.PrivateData.UpstreamHTTPTrace.PreparationRequest.URL, "/asset/seedance2/assetUpload")
	assert.Equal(t, "[REDACTED]", task.PrivateData.UpstreamHTTPTrace.PreparationRequest.Headers["Authorization"])
	assert.Contains(t, task.PrivateData.UpstreamHTTPTrace.PreparationRequest.Body, "https://media.example.com/bad.png")
	require.NotNil(t, task.PrivateData.UpstreamHTTPTrace.PreparationResponse)
	assert.Equal(t, http.StatusOK, task.PrivateData.UpstreamHTTPTrace.PreparationResponse.StatusCode)
	assert.Contains(t, task.PrivateData.UpstreamHTTPTrace.PreparationResponse.Body, "unsupported media")
}

func TestPrepareGlobalAIOpcTaskReturnsSafeUpstreamSubmissionError(t *testing.T) {
	service.InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/kyyReactApiServer/v2/model-center/tasks", request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"code":"HTTP_400_C400999","message":"[SY_ERR:60] HTTP 400: Duration must be between 1.8s and 30.2s."}}`)
	}))
	t.Cleanup(server.Close)

	task := &model.Task{PrivateData: model.TaskPrivateData{
		Key: "pinned-key",
		VideoPreparation: &model.VideoTaskPreparation{
			RequestBody: `{"model":"sd_2.5_special_v1","duration":31,"reference_images":["https://media.example.com/subject.png"]}`,
			DeadlineAt:  common.GetTimestamp() + 600,
			Assets: []model.VideoTaskAsset{{
				Field:     "reference_images",
				Index:     0,
				AssetType: "Image",
				SourceURL: "https://media.example.com/subject.png",
				AssetID:   "asset-image",
				Status:    "ACTIVE",
			}},
		},
	}}

	_, err := (&TaskAdaptor{}).PrepareTask(context.Background(), &model.Channel{BaseURL: common.GetPointer(server.URL)}, task)
	require.Error(t, err)
	var preparationErr *service.TaskPreparationError
	require.True(t, errors.As(err, &preparationErr))
	assert.Equal(t,
		"HTTP_400_C400999: [SY_ERR:60] HTTP 400: Duration must be between 1.8s and 30.2s.",
		preparationErr.PublicMessage,
	)
	assert.EqualError(t, err, "prepared GlobalAIOpc task endpoint returned HTTP 400")
	require.NotNil(t, task.PrivateData.UpstreamHTTPTrace)
	require.NotNil(t, task.PrivateData.UpstreamHTTPTrace.SubmitResponse)
	assert.Equal(t, http.StatusBadRequest, task.PrivateData.UpstreamHTTPTrace.SubmitResponse.StatusCode)
	assert.Contains(t, task.PrivateData.UpstreamHTTPTrace.SubmitResponse.Body, "Duration must be between")
}

func TestPrepareGlobalAIOpcTaskHonorsConfiguredOperationBatchSize(t *testing.T) {
	service.InitHttpClient()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests++
		_, _ = fmt.Fprintf(w, `{"code":0,"data":{"assetId":"asset-%d","materialStatus":"PROCESSING","status":"NONE"},"msg":null}`, requests)
	}))
	t.Cleanup(server.Close)

	channel := &model.Channel{BaseURL: common.GetPointer(server.URL)}
	channel.SetSetting(dto.ChannelSettings{
		VideoProtocol: dto.VideoProtocolGlobalAIOpc,
		GlobalAIOpcAssetPreparation: &dto.GlobalAIOpcAssetPreparationSettings{
			OperationsPerTaskPerPass: 2,
		},
	})
	task := &model.Task{PrivateData: model.TaskPrivateData{
		Key: "pinned-key",
		VideoPreparation: &model.VideoTaskPreparation{
			RequestBody: `{"model":"sd_2.5_special_v1","reference_images":["https://media.example.com/1.png","https://media.example.com/2.png","https://media.example.com/3.png"]}`,
			DeadlineAt:  common.GetTimestamp() + 600,
			Assets: []model.VideoTaskAsset{
				{Field: "reference_images", Index: 0, AssetType: "Image", SourceURL: "https://media.example.com/1.png"},
				{Field: "reference_images", Index: 1, AssetType: "Image", SourceURL: "https://media.example.com/2.png"},
				{Field: "reference_images", Index: 2, AssetType: "Image", SourceURL: "https://media.example.com/3.png"},
			},
		},
	}}

	result, err := (&TaskAdaptor{}).PrepareTask(context.Background(), channel, task)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Waiting)
	assert.Equal(t, 2, requests)
	assert.Equal(t, "asset-1", task.PrivateData.VideoPreparation.Assets[0].AssetID)
	assert.Equal(t, "asset-2", task.PrivateData.VideoPreparation.Assets[1].AssetID)
	assert.Empty(t, task.PrivateData.VideoPreparation.Assets[2].AssetID)
}
