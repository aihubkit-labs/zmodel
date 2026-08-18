package sora

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const (
	globalAIOpcAssetResponseLimit = 1 << 20
)

type globalAIOpcAssetResponse struct {
	Code    int                     `json:"code"`
	Data    *globalAIOpcAssetResult `json:"data"`
	Message string                  `json:"msg"`
}

type globalAIOpcAssetResult struct {
	AssetID      string `json:"assetId"`
	AssetType    string `json:"assetType"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
}

func (a *TaskAdaptor) BuildTaskPreparation(c *gin.Context, info *relaycommon.RelayInfo, requestBody []byte) (*model.VideoTaskPreparation, error) {
	if info == nil || info.ChannelSetting.VideoProtocol != dto.VideoProtocolGlobalAIOpc {
		return nil, nil
	}
	capability, ok := info.ChannelSetting.GetVideoModelCapability(info.UpstreamModelName)
	if !ok || capability.AssetPreparationMode != dto.VideoAssetPreparationGlobalAIOpcSeedance {
		return nil, nil
	}
	request, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	assets := make([]model.VideoTaskAsset, 0, len(request.ReferenceImages)+len(request.ReferenceVideos)+len(request.ReferenceAudios))
	for index, sourceURL := range request.ReferenceImages {
		assets = append(assets, model.VideoTaskAsset{Field: "reference_images", Index: index, AssetType: "Image", SourceURL: sourceURL})
	}
	for index, sourceURL := range request.ReferenceVideos {
		assets = append(assets, model.VideoTaskAsset{Field: "reference_videos", Index: index, AssetType: "Video", SourceURL: sourceURL})
	}
	for index, sourceURL := range request.ReferenceAudios {
		assets = append(assets, model.VideoTaskAsset{Field: "reference_audios", Index: index, AssetType: "Audio", SourceURL: sourceURL})
	}
	if len(assets) == 0 {
		return nil, nil
	}
	preparationSettings := info.ChannelSetting.GetGlobalAIOpcAssetPreparationSettings()
	return &model.VideoTaskPreparation{
		RequestBody: string(requestBody),
		Assets:      assets,
		DeadlineAt:  common.GetTimestamp() + int64(preparationSettings.TimeoutSeconds),
	}, nil
}

func (a *TaskAdaptor) PrepareTask(ctx context.Context, channel *model.Channel, task *model.Task) (*service.TaskPreparationResult, error) {
	if channel == nil || task == nil || task.PrivateData.VideoPreparation == nil {
		return nil, fmt.Errorf("missing GlobalAIOpc video preparation state")
	}
	preparation := task.PrivateData.VideoPreparation
	now := common.GetTimestamp()
	if preparation.DeadlineAt <= now {
		return nil, fmt.Errorf("GlobalAIOpc asset preparation timed out")
	}
	apiKey := strings.TrimSpace(task.PrivateData.Key)
	if apiKey == "" {
		return nil, fmt.Errorf("missing pinned GlobalAIOpc credential")
	}
	channelSetting := channel.GetSetting()
	preparationSettings := channelSetting.GetGlobalAIOpcAssetPreparationSettings()
	trace := task.PrivateData.UpstreamHTTPTrace
	if trace == nil {
		trace = &dto.TaskUpstreamHTTPTrace{}
		task.PrivateData.UpstreamHTTPTrace = trace
	}

	operations := 0
	for index := range preparation.Assets {
		asset := &preparation.Assets[index]
		status := strings.ToUpper(strings.TrimSpace(asset.Status))
		if status == "ACTIVE" {
			continue
		}
		if status == "FAILED" || status == "DELETED" {
			return nil, fmt.Errorf("GlobalAIOpc asset %s entered terminal status %s", asset.AssetID, status)
		}
		if operations >= preparationSettings.OperationsPerTaskPerPass || asset.NextPollAt > now {
			continue
		}

		endpoint := taskcommon.BuildGlobalAIOpcAssetDetailURL(channel.GetBaseURL())
		payload := map[string]any{"assetId": asset.AssetID}
		if strings.TrimSpace(asset.AssetID) == "" {
			endpoint = taskcommon.BuildGlobalAIOpcAssetUploadURL(channel.GetBaseURL())
			payload = map[string]any{
				"assetType": asset.AssetType,
				"url":       asset.SourceURL,
			}
		}
		statusCode, responseBody, requestErr := doGlobalAIOpcJSON(ctx, channelSetting.Proxy, apiKey, endpoint, payload, trace)
		asset.Attempts++
		asset.NextPollAt = now + globalAIOpcAssetRetryDelay(asset.Attempts)
		operations++
		if requestErr != nil || statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError {
			continue
		}
		if statusCode != http.StatusOK {
			internalErr := fmt.Errorf("GlobalAIOpc asset endpoint returned HTTP %d", statusCode)
			if publicMessage := globalAIOpcPublicError(responseBody); publicMessage != "" {
				return nil, service.NewTaskPreparationError(publicMessage, internalErr)
			}
			return nil, internalErr
		}
		var response globalAIOpcAssetResponse
		if err := common.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("decode GlobalAIOpc asset response: %w", err)
		}
		if response.Code != 0 {
			internalErr := fmt.Errorf("GlobalAIOpc asset endpoint returned code %d: %s", response.Code, strings.TrimSpace(response.Message))
			publicMessage := strings.TrimSpace(response.Message)
			if publicMessage != "" {
				publicMessage = fmt.Sprintf("%d: %s", response.Code, publicMessage)
				return nil, service.NewTaskPreparationError(relaycommon.SanitizeUpstreamHTTPText(publicMessage), internalErr)
			}
			return nil, internalErr
		}
		if response.Data == nil {
			return nil, fmt.Errorf("GlobalAIOpc asset response omitted data")
		}
		if strings.TrimSpace(response.Data.AssetID) == "" {
			return nil, fmt.Errorf("GlobalAIOpc asset response omitted assetId")
		}
		if asset.AssetID != "" && asset.AssetID != response.Data.AssetID {
			return nil, fmt.Errorf("GlobalAIOpc asset detail returned a different assetId")
		}
		asset.AssetID = response.Data.AssetID
		asset.Status = strings.ToUpper(strings.TrimSpace(response.Data.Status))
		asset.ErrorMessage = strings.TrimSpace(response.Data.ErrorMessage)
		if asset.ErrorMessage == "" {
			asset.ErrorMessage = strings.TrimSpace(response.Message)
		}
		if asset.Status == "FAILED" || asset.Status == "DELETED" {
			internalErr := fmt.Errorf("GlobalAIOpc asset %s entered terminal status %s: %s", asset.AssetID, asset.Status, asset.ErrorMessage)
			if asset.ErrorMessage != "" {
				return nil, service.NewTaskPreparationError(relaycommon.SanitizeUpstreamHTTPText(asset.ErrorMessage), internalErr)
			}
			return nil, internalErr
		}
	}

	for _, asset := range preparation.Assets {
		if strings.ToUpper(strings.TrimSpace(asset.Status)) != "ACTIVE" {
			return &service.TaskPreparationResult{Waiting: true}, nil
		}
	}
	requestBody, err := rewriteGlobalAIOpcAssetReferences(preparation)
	if err != nil {
		return nil, err
	}
	return submitPreparedGlobalAIOpcTask(ctx, channel, task, apiKey, requestBody)
}

func globalAIOpcAssetRetryDelay(attempts int) int64 {
	delay := int64(5)
	for current := 1; current < attempts && delay < 30; current++ {
		delay *= 2
	}
	if delay > 30 {
		return 30
	}
	return delay
}

func globalAIOpcPublicError(responseBody []byte) string {
	var response struct {
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
		Msg     string          `json:"msg"`
		Error   json.RawMessage `json:"error"`
	}
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return ""
	}

	message := strings.TrimSpace(response.Message)
	if message == "" {
		message = strings.TrimSpace(response.Msg)
	}
	code := ""
	if upstreamError := publicVideoError(response.Error); upstreamError != nil {
		if message == "" {
			message = strings.TrimSpace(upstreamError.Message)
		}
		code = strings.TrimSpace(upstreamError.Code)
	}
	if code == "" && len(response.Code) > 0 && string(response.Code) != "null" {
		var codeValue any
		if common.Unmarshal(response.Code, &codeValue) == nil {
			code = strings.TrimSpace(fmt.Sprint(codeValue))
		}
	}
	if message == "" {
		return ""
	}
	if code != "" && code != "0" && code != "upstream_error" && !strings.Contains(message, code) {
		message = code + ": " + message
	}
	return relaycommon.SanitizeUpstreamHTTPText(message)
}

func doGlobalAIOpcJSON(ctx context.Context, proxy, apiKey, endpoint string, payload any, trace *dto.TaskUpstreamHTTPTrace) (int, []byte, error) {
	body, err := common.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	trace.PreparationRequest = relaycommon.CaptureUpstreamHTTPRequest(request)
	trace.PreparationResponse = nil
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		trace.PreparationResponse = relaycommon.UpstreamHTTPTransportError(err)
		return 0, nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		trace.PreparationResponse = relaycommon.UpstreamHTTPTransportError(err)
		return 0, nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, globalAIOpcAssetResponseLimit))
	trace.PreparationResponse = relaycommon.UpstreamHTTPResponseFromBody(response, responseBody)
	if err != nil {
		trace.PreparationResponse.Error = relaycommon.UpstreamHTTPTransportError(err).Error
	}
	return response.StatusCode, responseBody, err
}

func rewriteGlobalAIOpcAssetReferences(preparation *model.VideoTaskPreparation) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := common.Unmarshal([]byte(preparation.RequestBody), &fields); err != nil {
		return nil, fmt.Errorf("decode prepared GlobalAIOpc request: %w", err)
	}
	valuesByField := make(map[string][]string)
	for _, asset := range preparation.Assets {
		values, ok := valuesByField[asset.Field]
		if !ok {
			if err := common.Unmarshal(fields[asset.Field], &values); err != nil {
				return nil, fmt.Errorf("decode prepared field %s: %w", asset.Field, err)
			}
		}
		if asset.Index < 0 || asset.Index >= len(values) {
			return nil, fmt.Errorf("prepared field %s index %d is out of range", asset.Field, asset.Index)
		}
		values[asset.Index] = "assetId://" + asset.AssetID
		valuesByField[asset.Field] = values
	}
	for field, values := range valuesByField {
		encoded, err := common.Marshal(values)
		if err != nil {
			return nil, err
		}
		fields[field] = encoded
	}
	return common.Marshal(fields)
}

func submitPreparedGlobalAIOpcTask(ctx context.Context, channel *model.Channel, task *model.Task, apiKey string, requestBody []byte) (*service.TaskPreparationResult, error) {
	endpoint := taskcommon.BuildGlobalAIOpcVideoTaskURL(channel.GetBaseURL(), "")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	trace := task.PrivateData.UpstreamHTTPTrace
	if trace == nil {
		trace = &dto.TaskUpstreamHTTPTrace{}
	}
	trace.SubmitRequest = relaycommon.CaptureUpstreamHTTPRequest(request)
	trace.SubmitResponse = nil
	task.PrivateData.UpstreamHTTPTrace = trace
	client, err := service.GetHttpClientWithProxy(channel.GetSetting().Proxy)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		trace.SubmitResponse = relaycommon.UpstreamHTTPTransportError(err)
		task.PrivateData.UpstreamHTTPTrace = trace
		return nil, fmt.Errorf("submit prepared GlobalAIOpc task: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, globalAIOpcAssetResponseLimit))
	trace.SubmitResponse = relaycommon.UpstreamHTTPResponseFromBody(response, responseBody)
	task.PrivateData.UpstreamHTTPTrace = trace
	if err != nil {
		return nil, fmt.Errorf("read prepared GlobalAIOpc task response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		internalErr := fmt.Errorf("prepared GlobalAIOpc task endpoint returned HTTP %d", response.StatusCode)
		if publicMessage := globalAIOpcPublicError(responseBody); publicMessage != "" {
			return nil, service.NewTaskPreparationError(publicMessage, internalErr)
		}
		return nil, internalErr
	}
	var upstream responseTask
	if err := common.Unmarshal(responseBody, &upstream); err != nil {
		return nil, fmt.Errorf("decode prepared GlobalAIOpc task response: %w", err)
	}
	upstreamTaskID := strings.TrimSpace(upstream.ID)
	if upstreamTaskID == "" {
		upstreamTaskID = strings.TrimSpace(upstream.TaskID)
	}
	if upstreamTaskID == "" {
		internalErr := fmt.Errorf("prepared GlobalAIOpc task response omitted task ID")
		if publicMessage := globalAIOpcPublicError(responseBody); publicMessage != "" {
			return nil, service.NewTaskPreparationError(publicMessage, internalErr)
		}
		return nil, internalErr
	}
	storedBody, err := sanitizeStoredVideoResponse(responseBody)
	if err != nil {
		return nil, err
	}
	return &service.TaskPreparationResult{
		UpstreamTaskID:    upstreamTaskID,
		TaskData:          storedBody,
		UpstreamHTTPTrace: trace,
	}, nil
}
