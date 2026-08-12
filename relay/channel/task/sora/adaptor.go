package sora

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

type responseTask struct {
	ID             string          `json:"id"`
	TaskID         string          `json:"task_id,omitempty"` //兼容旧接口
	Object         string          `json:"object"`
	Model          string          `json:"model"`
	Status         string          `json:"status"`
	Progress       int             `json:"progress"`
	Created        int64           `json:"created,omitempty"`
	CreatedAt      int64           `json:"created_at"`
	CompletedAt    int64           `json:"completed_at,omitempty"`
	ExpiresAt      int64           `json:"expires_at,omitempty"`
	Seconds        string          `json:"seconds,omitempty"`
	Duration       json.RawMessage `json:"duration,omitempty"`
	Size           string          `json:"size,omitempty"`
	Resolution     string          `json:"resolution,omitempty"`
	Ratio          string          `json:"ratio,omitempty"`
	URL            string          `json:"url,omitempty"`
	ResultURL      string          `json:"result_url,omitempty"`
	VideoURL       string          `json:"video_url,omitempty"`
	ActualDuration json.RawMessage `json:"actualDuration,omitempty"`
	Message        string          `json:"message,omitempty"`
	Msg            string          `json:"msg,omitempty"`
	Metadata       *struct {
		SizeMapping struct {
			Resolution string `json:"resolution,omitempty"`
			Ratio      string `json:"ratio,omitempty"`
		} `json:"size_mapping,omitempty"`
	} `json:"metadata,omitempty"`
	RemixedFromVideoID string          `json:"remixed_from_video_id,omitempty"`
	Error              json.RawMessage `json:"error,omitempty"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func validateRemixRequest(c *gin.Context) *dto.TaskError {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("field prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	// 存储原始请求到 context，与 ValidateMultipartDirect 路径保持一致
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if info.Action == constant.TaskActionRemix {
		if dto.IsGlobalAIOpcVideoProtocol(info.ChannelSetting.VideoProtocol) {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("video remix is not supported by this model"),
				"unsupported_operation",
				http.StatusBadRequest,
			)
		}
		if taskErr := validateRemixRequest(c); taskErr != nil {
			return taskErr
		}
		return validateVideoProtocolRequest(c, info)
	}
	if dto.IsGlobalAIOpcVideoProtocol(info.ChannelSetting.VideoProtocol) &&
		!strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return videoRequestError("this video model requires an application/json request with public media URLs", "unsupported_content_type")
	}
	if info.ChannelSetting.VideoProtocol == dto.VideoProtocolAgnesVideoV2 &&
		strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		if taskErr := validateAgnesMultipartReferenceInput(c); taskErr != nil {
			return taskErr
		}
	}
	if taskErr := relaycommon.ValidateMultipartDirect(c, info); taskErr != nil {
		return taskErr
	}
	if taskErr := validateVideoProtocolRequest(c, info); taskErr != nil {
		return taskErr
	}
	if taskErr := normalizeVideoProtocolRequest(c, info); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	if taskErr := validateVideoModelCapability(info, req); taskErr != nil {
		return taskErr
	}
	if !usesConfiguredDurationVideoProtocol(info) {
		return nil
	}
	duration := req.Duration
	if duration == 0 && req.Seconds != "" {
		duration, err = strconv.Atoi(req.Seconds)
		if err != nil {
			return videoParameterError(
				fmt.Sprintf("duration must be an integer; received %q", req.Seconds),
				"invalid_seconds",
				dto.VideoParameterErrorData{Parameter: "duration", Received: req.Seconds},
			)
		}
	}
	if duration == 0 && !info.VideoDurationRequired {
		return nil
	}
	if duration < info.VideoMinDurationSeconds || duration > info.VideoMaxDurationSeconds {
		return videoIntegerRangeError(
			fmt.Sprintf("duration must be between %d and %d seconds; received %d", info.VideoMinDurationSeconds, info.VideoMaxDurationSeconds, duration),
			"invalid_seconds",
			"duration",
			int64(duration),
			int64(info.VideoMinDurationSeconds),
			int64(info.VideoMaxDurationSeconds),
		)
	}
	req.Duration = duration
	req.Seconds = ""
	req.Resolution = strings.ToLower(strings.TrimSpace(req.Resolution))
	c.Set("task_request", req)
	return nil
}

func usesConfiguredDurationVideoProtocol(info *relaycommon.RelayInfo) bool {
	return info != nil && info.VideoMinDurationSeconds > 0 && info.VideoMaxDurationSeconds >= info.VideoMinDurationSeconds
}

func videoResolutionSupported(capability dto.VideoModelCapability, resolution string) bool {
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	for _, supportedResolution := range capability.Resolutions {
		if strings.ToLower(strings.TrimSpace(supportedResolution)) == resolution {
			return true
		}
	}
	return false
}

func (a *TaskAdaptor) EstimateBillingDimensions(c *gin.Context, info *relaycommon.RelayInfo) (billingexpr.BillingDimensions, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return billingexpr.BillingDimensions{}, err
	}
	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if usesConfiguredDurationVideoProtocol(info) {
		if seconds == 0 && !info.VideoDurationRequired {
			return billingexpr.BillingDimensions{Units: 1, ResolutionTier: resolution}, nil
		}
		if seconds < info.VideoMinDurationSeconds || seconds > info.VideoMaxDurationSeconds {
			return billingexpr.BillingDimensions{}, fmt.Errorf(
				"duration is required and must be between %d and %d seconds for tiered billing",
				info.VideoMinDurationSeconds,
				info.VideoMaxDurationSeconds,
			)
		}
		if !videoResolutionSupported(dto.VideoModelCapability{Resolutions: info.VideoAllowedResolutions}, resolution) {
			return billingexpr.BillingDimensions{}, fmt.Errorf("invalid resolution %s for configured video model", resolution)
		}
	}
	if seconds <= 0 {
		return billingexpr.BillingDimensions{}, nil
	}
	if resolution == "" {
		resolution = strings.ToLower(strings.TrimSpace(req.Size))
	}
	return billingexpr.BillingDimensions{
		Units:          1,
		Seconds:        float64(seconds),
		ResolutionTier: resolution,
	}, nil
}

// EstimateBilling 根据用户请求的 seconds 和 size 计算 OtherRatios。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	// remix 路径的 OtherRatios 已在 ResolveOriginTask 中设置
	if info.Action == constant.TaskActionRemix {
		return nil
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds <= 0 {
		seconds = 4
	}

	size := req.Size
	if size == "" {
		size = "720x1280"
	}

	ratios := map[string]float64{
		"seconds": float64(seconds),
		"size":    1,
	}
	if size == "1792x1024" || size == "1024x1792" {
		ratios["size"] = 1.666667
	}
	return ratios
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimRight(a.baseURL, "/")
	if dto.IsGlobalAIOpcVideoProtocol(info.ChannelSetting.VideoProtocol) {
		if info.Action == constant.TaskActionRemix {
			return "", fmt.Errorf("video remix is not supported by this model")
		}
		return taskcommon.BuildGlobalAIOpcVideoTaskURL(baseURL, ""), nil
	}
	if info.Action == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", baseURL, info.OriginTaskID), nil
	}
	return baseURL + "/v1/videos", nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	if info != nil && info.ChannelSetting.VideoProtocol == dto.VideoProtocolMegabyAI {
		if strings.TrimSpace(info.PublicTaskID) == "" {
			return fmt.Errorf("public task ID is required for MegabyAI idempotency")
		}
		req.Header.Set("Idempotency-Key", "zmodel:"+info.PublicTaskID)
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]json.RawMessage
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			modelJSON, marshalErr := common.Marshal(info.UpstreamModelName)
			if marshalErr != nil {
				return nil, errors.Wrap(marshalErr, "marshal_upstream_model_failed")
			}
			bodyMap["model"] = modelJSON
			if parsed, getErr := relaycommon.GetTaskRequest(c); getErr == nil {
				if usesConfiguredDurationVideoProtocol(info) {
					delete(bodyMap, "seconds")
					resolutionJSON, marshalErr := common.Marshal(parsed.Resolution)
					if marshalErr != nil {
						return nil, errors.Wrap(marshalErr, "marshal_resolution_failed")
					}
					bodyMap["resolution"] = resolutionJSON
					if usesExtendedVideoModelCapabilities(info.ChannelSetting.VideoProtocol) {
						ratioJSON, marshalErr := common.Marshal(parsed.Ratio)
						if marshalErr != nil {
							return nil, errors.Wrap(marshalErr, "marshal_ratio_failed")
						}
						bodyMap["ratio"] = ratioJSON
					}
					if parsed.Duration > 0 {
						durationJSON, marshalErr := common.Marshal(parsed.Duration)
						if marshalErr != nil {
							return nil, errors.Wrap(marshalErr, "marshal_duration_failed")
						}
						bodyMap["duration"] = durationJSON
					}
				}
				capability, _ := info.ChannelSetting.GetVideoModelCapability(info.UpstreamModelName)
				if err := applyVideoProtocolRequest(info.ChannelSetting.VideoProtocol, capability, parsed, bodyMap); err != nil {
					return nil, errors.Wrap(err, "normalize_video_protocol_request_failed")
				}
			}
			mergeVideoProviderOptions(c, bodyMap, info.ChannelSetting.VideoProtocol)
			if newBody, err := common.Marshal(bodyMap); err == nil {
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("model", info.UpstreamModelName)
		parsed, _ := relaycommon.GetTaskRequest(c)
		megabyAIRequest := info.ChannelSetting.VideoProtocol == dto.VideoProtocolMegabyAI
		agnesRequest := info.ChannelSetting.VideoProtocol == dto.VideoProtocolAgnesVideoV2
		for key, values := range formData.Value {
			if key == "model" {
				continue
			}
			if megabyAIRequest && (key == "resolution" || key == "duration" || key == "seconds" ||
				(info.ChannelSetting.VideoProtocol == dto.VideoProtocolMegabyAI && key == "ratio")) {
				continue
			}
			if agnesRequest && (key == "duration" || key == "seconds" || key == "num_frames" || key == "frame_rate" ||
				key == "resolution" || key == "ratio" || key == "size" || key == "width" || key == "height") {
				continue
			}
			for _, v := range values {
				writer.WriteField(key, v)
			}
		}
		if megabyAIRequest {
			writer.WriteField("resolution", parsed.Resolution)
		}
		if info.ChannelSetting.VideoProtocol == dto.VideoProtocolMegabyAI {
			writer.WriteField("ratio", parsed.Ratio)
		}
		if megabyAIRequest && parsed.Duration > 0 {
			writer.WriteField("duration", strconv.Itoa(parsed.Duration))
		}
		if agnesRequest && parsed.Duration > 0 {
			numFrames, frameRate := agnesVideoFrameParameters(parsed.Duration)
			writer.WriteField("num_frames", strconv.Itoa(numFrames))
			writer.WriteField("frame_rate", strconv.Itoa(frameRate))
			if width, height, ok := agnesVideoDimensions(parsed.Resolution, parsed.Ratio); ok {
				writer.WriteField("width", strconv.Itoa(width))
				writer.WriteField("height", strconv.Itoa(height))
			}
		}
		for fieldName, fileHeaders := range formData.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					// Re-open after sniffing so the full content is copied below
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}
		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return common.ReaderOnly(storage), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse Sora response
	var dResp responseTask
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	upstreamID := dResp.ID
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	if upstreamID == "" {
		upstreamMessage := strings.TrimSpace(dResp.Message)
		if upstreamMessage == "" {
			upstreamMessage = strings.TrimSpace(dResp.Msg)
		}
		if upstreamMessage == "" {
			if upstreamError := publicVideoError(dResp.Error); upstreamError != nil {
				upstreamMessage = strings.TrimSpace(upstreamError.Message)
			}
		}
		if upstreamMessage != "" {
			taskErr = service.TaskErrorWrapper(
				fmt.Errorf("upstream rejected the video task request"),
				"upstream_request_failed",
				http.StatusBadGateway,
			)
			return
		}
		taskErr = service.TaskErrorWrapper(fmt.Errorf("upstream response does not contain a task ID"), "invalid_response", http.StatusBadGateway)
		return
	}

	request, _ := relaycommon.GetTaskRequest(c)
	publicModel := request.Model
	if publicModel == "" {
		publicModel = info.OriginModelName
	}
	createdAt := dResp.CreatedAt
	if createdAt == 0 {
		createdAt = dResp.Created
	}
	duration := responseDuration(dResp.ActualDuration, "")
	if duration == 0 {
		duration = responseDuration(dResp.Duration, dResp.Seconds)
	}
	if duration == 0 {
		duration = request.Duration
	}
	publicBody, err := buildPublicVideoResponse(publicVideoResponseValues{
		ID:          info.PublicTaskID,
		Model:       publicModel,
		Status:      normalizePublicVideoStatus(dResp.Status),
		Progress:    dResp.Progress,
		CreatedAt:   createdAt,
		CompletedAt: dResp.CompletedAt,
		ExpiresAt:   dResp.ExpiresAt,
		Duration:    duration,
		Resolution:  request.Resolution,
		Ratio:       request.Ratio,
		Size:        dResp.Size,
		Error:       publicVideoError(dResp.Error),
	})
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "normalize_video_response_failed", http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", publicBody)
	storedBody, err := sanitizeStoredVideoResponse(responseBody)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "sanitize_video_response_failed", http.StatusInternalServerError)
		return
	}
	return upstreamID, storedBody, nil
}

func sanitizeStoredVideoResponse(responseBody []byte) ([]byte, error) {
	var payload map[string]any
	if err := common.Unmarshal(responseBody, &payload); err != nil {
		return nil, err
	}
	for _, key := range []string{"url", "result_url", "video_url"} {
		delete(payload, key)
	}
	if metadata, ok := payload["metadata"].(map[string]any); ok {
		for _, key := range []string{"url", "content_url", "local_url", "video_url", "final_video_url", "origin_video_url"} {
			delete(metadata, key)
		}
	}
	return common.Marshal(payload)
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	baseUrl = strings.TrimRight(baseUrl, "/")
	protocol := dto.VideoProtocol(strings.TrimSpace(fmt.Sprint(body["video_protocol"])))
	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)
	if dto.IsGlobalAIOpcVideoProtocol(protocol) {
		uri = taskcommon.BuildGlobalAIOpcVideoTaskURL(baseUrl, taskID)
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, relaycommon.WrapUpstreamRequestError(req, fmt.Errorf("new proxy http client failed: %w", err))
	}
	return relaycommon.DoTaskRequest(client, req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	switch resTask.Status {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "completed":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Url = strings.TrimSpace(resTask.ResultURL)
		if taskResult.Url == "" {
			taskResult.Url = strings.TrimSpace(resTask.VideoURL)
		}
		if taskResult.Url == "" {
			taskResult.Url = strings.TrimSpace(resTask.URL)
		}
	case "failed", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if publicError := publicVideoError(resTask.Error); publicError != nil {
			taskResult.Reason = publicError.Message
		} else {
			taskResult.Reason = "task failed"
		}
	default:
	}
	taskResult.Duration = responseDuration(resTask.ActualDuration, "")
	if taskResult.Duration == 0 {
		taskResult.Duration = responseDuration(resTask.Duration, resTask.Seconds)
	}
	taskResult.Resolution = strings.ToLower(strings.TrimSpace(resTask.Resolution))
	if taskResult.Resolution == "" && resTask.Metadata != nil {
		taskResult.Resolution = strings.ToLower(strings.TrimSpace(resTask.Metadata.SizeMapping.Resolution))
	}
	if taskResult.Resolution == "" {
		taskResult.Resolution = strings.ToLower(strings.TrimSpace(resTask.Size))
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", resTask.Progress)
	}

	return &taskResult, nil
}

func responseDuration(duration json.RawMessage, seconds string) int {
	if len(duration) > 0 && string(duration) != "null" {
		var numeric int64
		if err := common.Unmarshal(duration, &numeric); err == nil && numeric > 0 && numeric <= relaycommon.MaxTaskDurationSeconds {
			return int(numeric)
		}
		var text string
		if err := common.Unmarshal(duration, &text); err == nil {
			if numeric, err = strconv.ParseInt(text, 10, 64); err == nil && numeric > 0 && numeric <= relaycommon.MaxTaskDurationSeconds {
				return int(numeric)
			}
		}
	}
	numericSeconds, err := strconv.ParseFloat(seconds, 64)
	if err != nil || math.IsNaN(numericSeconds) || math.IsInf(numericSeconds, 0) ||
		numericSeconds <= 0 || numericSeconds > relaycommon.MaxTaskDurationSeconds {
		return 0
	}
	return int(math.Round(numericSeconds))
}

type publicVideoResponseValues struct {
	ID                 string
	Model              string
	Status             string
	Progress           int
	CreatedAt          int64
	CompletedAt        int64
	ExpiresAt          int64
	Duration           int
	Resolution         string
	Ratio              string
	Size               string
	RemixedFromVideoID string
	Error              *dto.OpenAIVideoError
}

func normalizePublicVideoStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "submitted", "queued", "pending":
		return dto.VideoStatusQueued
	case "processing", "in_progress", "running":
		return dto.VideoStatusInProgress
	case "completed", "success", "succeeded":
		return dto.VideoStatusCompleted
	case "failed", "failure", "cancelled", "canceled":
		return dto.VideoStatusFailed
	case "unknown":
		return dto.VideoStatusUnknown
	default:
		return status
	}
}

func buildPublicVideoResponse(values publicVideoResponseValues) ([]byte, error) {
	video := dto.OpenAIVideo{
		ID:                 values.ID,
		TaskID:             values.ID,
		Object:             "video",
		Model:              values.Model,
		Status:             values.Status,
		Progress:           values.Progress,
		CreatedAt:          values.CreatedAt,
		CompletedAt:        values.CompletedAt,
		ExpiresAt:          values.ExpiresAt,
		Duration:           values.Duration,
		Resolution:         strings.ToLower(strings.TrimSpace(values.Resolution)),
		Ratio:              strings.TrimSpace(values.Ratio),
		Size:               strings.TrimSpace(values.Size),
		RemixedFromVideoID: values.RemixedFromVideoID,
		Error:              values.Error,
	}
	if values.Duration > 0 {
		video.Seconds = strconv.Itoa(values.Duration)
	}
	if values.Status == dto.VideoStatusCompleted && values.ID != "" {
		video.URL = taskcommon.BuildProxyURL(values.ID)
		video.VideoURL = video.URL
	}
	return common.Marshal(video)
}

func publicVideoError(raw json.RawMessage) *dto.OpenAIVideoError {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var structured dto.OpenAIVideoError
	if err := common.Unmarshal(raw, &structured); err == nil && strings.TrimSpace(structured.Message) != "" {
		return &structured
	}
	var message string
	if err := common.Unmarshal(raw, &message); err == nil && strings.TrimSpace(message) != "" {
		return &dto.OpenAIVideoError{Message: message, Code: "upstream_error"}
	}
	return &dto.OpenAIVideoError{Message: "video generation failed", Code: "upstream_error"}
}

func (a *TaskAdaptor) AdjustBillingDimensionsOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) *billingexpr.BillingDimensions {
	if taskResult == nil {
		return nil
	}
	protocol := dto.VideoProtocol("")
	if task != nil && task.PrivateData.BillingContext != nil {
		protocol = task.PrivateData.BillingContext.VideoProtocol
	}
	dimensions := billingexpr.BillingDimensions{}
	validDuration := taskResult.Duration > 0 && taskResult.Duration <= relaycommon.MaxTaskDurationSeconds
	if protocol == dto.VideoProtocolMegabyAI || protocol == dto.VideoProtocolGlobalAIOpc {
		validDuration = task != nil && task.PrivateData.BillingContext != nil &&
			task.PrivateData.BillingContext.VideoMinDurationSeconds > 0 &&
			task.PrivateData.BillingContext.VideoMaxDurationSeconds >= task.PrivateData.BillingContext.VideoMinDurationSeconds &&
			taskResult.Duration >= task.PrivateData.BillingContext.VideoMinDurationSeconds &&
			taskResult.Duration <= task.PrivateData.BillingContext.VideoMaxDurationSeconds
	}
	if protocol == dto.VideoProtocolAgnesVideoV2 {
		validDuration = taskResult.Duration >= 1 && taskResult.Duration <= agnesMaxDurationSeconds
	}
	if validDuration {
		dimensions.Seconds = float64(taskResult.Duration)
	}
	resolution := strings.ToLower(strings.TrimSpace(taskResult.Resolution))
	if resolution != "" && task != nil && task.PrivateData.BillingContext != nil &&
		videoResolutionSupported(dto.VideoModelCapability{
			Resolutions: task.PrivateData.BillingContext.VideoAllowedResolutions,
		}, resolution) {
		dimensions.ResolutionTier = resolution
	}
	if dimensions.Seconds == 0 && dimensions.ResolutionTier == "" {
		return nil
	}
	return &dimensions
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var upstream responseTask
	if err := common.Unmarshal(task.Data, &upstream); err != nil {
		return nil, errors.Wrap(err, "unmarshal stored video response failed")
	}

	request := relaycommon.TaskRequestSnapshot{}
	if task.Properties.Input != "" {
		_ = common.UnmarshalJsonStr(task.Properties.Input, &request)
	}

	publicModel := request.Model
	if publicModel == "" {
		publicModel = task.Properties.OriginModelName
	}
	status := normalizePublicVideoStatus(upstream.Status)
	if taskStatus := task.Status.ToVideoStatus(); taskStatus != dto.VideoStatusUnknown {
		status = taskStatus
	}
	progress := upstream.Progress
	if task.Progress != "" {
		progressText := strings.TrimSuffix(strings.TrimSpace(task.Progress), "%")
		if parsedProgress, err := strconv.Atoi(progressText); err == nil {
			progress = parsedProgress
		}
	}
	if task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure {
		progress = 100
	}
	duration := responseDuration(upstream.ActualDuration, "")
	if duration == 0 {
		duration = responseDuration(upstream.Duration, upstream.Seconds)
	}
	if duration == 0 {
		duration = request.Duration
		if duration == 0 && request.Seconds != "" {
			duration, _ = strconv.Atoi(request.Seconds)
		}
	}
	resolution := strings.ToLower(strings.TrimSpace(upstream.Resolution))
	if resolution == "" && upstream.Metadata != nil {
		resolution = strings.ToLower(strings.TrimSpace(upstream.Metadata.SizeMapping.Resolution))
	}
	if resolution == "" && task.PrivateData.BillingContext != nil &&
		videoResolutionSupported(dto.VideoModelCapability{
			Resolutions: task.PrivateData.BillingContext.VideoAllowedResolutions,
		}, upstream.Size) {
		resolution = strings.ToLower(strings.TrimSpace(upstream.Size))
	}
	if resolution == "" || strings.Contains(resolution, "x") {
		resolution = strings.ToLower(strings.TrimSpace(request.Resolution))
	}
	if dto.IsGlobalAIOpcVideoProtocol(task.GetVideoProtocol()) && strings.TrimSpace(request.Resolution) != "" {
		resolution = strings.ToLower(strings.TrimSpace(request.Resolution))
	}
	ratio := strings.TrimSpace(upstream.Ratio)
	if ratio == "" && upstream.Metadata != nil {
		ratio = strings.TrimSpace(upstream.Metadata.SizeMapping.Ratio)
	}
	if ratio == "" {
		ratio = strings.TrimSpace(request.Ratio)
	}
	size := strings.TrimSpace(upstream.Size)
	if size == "" && task.PrivateData.BillingContext != nil &&
		task.PrivateData.BillingContext.VideoProtocol == dto.VideoProtocolAgnesVideoV2 {
		if width, height, ok := agnesVideoDimensions(resolution, ratio); ok {
			size = fmt.Sprintf("%dx%d", width, height)
		}
	}
	completedAt := upstream.CompletedAt
	if task.FinishTime > 0 {
		completedAt = task.FinishTime
	}

	createdAt := upstream.CreatedAt
	if createdAt == 0 {
		createdAt = upstream.Created
	}
	if task.SubmitTime > 0 {
		createdAt = task.SubmitTime
	}
	publicError := publicVideoError(upstream.Error)
	if task.Status == model.TaskStatusFailure && publicError == nil && strings.TrimSpace(task.FailReason) != "" {
		publicError = &dto.OpenAIVideoError{Message: task.FailReason, Code: "video_generation_failed"}
	}
	data, err := buildPublicVideoResponse(publicVideoResponseValues{
		ID:                 task.TaskID,
		Model:              publicModel,
		Status:             status,
		Progress:           progress,
		CreatedAt:          createdAt,
		CompletedAt:        completedAt,
		ExpiresAt:          upstream.ExpiresAt,
		Duration:           duration,
		Resolution:         resolution,
		Ratio:              ratio,
		Size:               size,
		RemixedFromVideoID: upstream.RemixedFromVideoID,
		Error:              publicError,
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}
