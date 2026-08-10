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
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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
	ID          string          `json:"id"`
	TaskID      string          `json:"task_id,omitempty"` //兼容旧接口
	Object      string          `json:"object"`
	Model       string          `json:"model"`
	Status      string          `json:"status"`
	Progress    int             `json:"progress"`
	CreatedAt   int64           `json:"created_at"`
	CompletedAt int64           `json:"completed_at,omitempty"`
	ExpiresAt   int64           `json:"expires_at,omitempty"`
	Seconds     string          `json:"seconds,omitempty"`
	Duration    json.RawMessage `json:"duration,omitempty"`
	Size        string          `json:"size,omitempty"`
	Resolution  string          `json:"resolution,omitempty"`
	Ratio       string          `json:"ratio,omitempty"`
	Metadata    *struct {
		SizeMapping struct {
			Resolution string `json:"resolution,omitempty"`
			Ratio      string `json:"ratio,omitempty"`
		} `json:"size_mapping,omitempty"`
	} `json:"metadata,omitempty"`
	RemixedFromVideoID string `json:"remixed_from_video_id,omitempty"`
	Error              *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
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
		if taskErr := validateRemixRequest(c); taskErr != nil {
			return taskErr
		}
		return validateVideoProtocolRequest(c, info)
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
			return service.TaskErrorWrapperLocal(fmt.Errorf("duration must be an integer"), "invalid_seconds", http.StatusBadRequest)
		}
	}
	if duration < info.VideoMinDurationSeconds || duration > info.VideoMaxDurationSeconds {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("duration must be between %d and %d seconds", info.VideoMinDurationSeconds, info.VideoMaxDurationSeconds),
			"invalid_seconds",
			http.StatusBadRequest,
		)
	}
	req.Duration = duration
	req.Seconds = ""
	req.Resolution = strings.ToLower(strings.TrimSpace(req.Resolution))
	c.Set("task_request", req)
	return nil
}

func usesConfiguredDurationVideoProtocol(info *relaycommon.RelayInfo) bool {
	return info != nil && videoProtocolUsesConfiguredDuration(info.ChannelSetting.VideoProtocol)
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
	if info.Action == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", a.baseURL, info.OriginTaskID), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	if info != nil && info.ChannelSetting.VideoProtocol == dto.VideoProtocolMinimaxH3MegabyAI {
		if strings.TrimSpace(info.PublicTaskID) == "" {
			return fmt.Errorf("public task ID is required for MiniMax H3 idempotency")
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
					if info.ChannelSetting.VideoProtocol == dto.VideoProtocolMinimaxH3MegabyAI {
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
				if err := applyVideoProtocolRequest(info.ChannelSetting.VideoProtocol, parsed, bodyMap); err != nil {
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
		megabyAIRequest := usesConfiguredDurationVideoProtocol(info)
		agnesRequest := info.ChannelSetting.VideoProtocol == dto.VideoProtocolAgnesVideoV2
		for key, values := range formData.Value {
			if key == "model" {
				continue
			}
			if megabyAIRequest && (key == "resolution" || key == "duration" || key == "seconds" ||
				(info.ChannelSetting.VideoProtocol == dto.VideoProtocolMinimaxH3MegabyAI && key == "ratio")) {
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
		if info.ChannelSetting.VideoProtocol == dto.VideoProtocolMinimaxH3MegabyAI {
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
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	request, _ := relaycommon.GetTaskRequest(c)
	publicModel := request.Model
	if publicModel == "" {
		publicModel = info.OriginModelName
	}
	publicBody, err := normalizePublicVideoResponse(responseBody, publicVideoResponseValues{
		ID:          info.PublicTaskID,
		Model:       publicModel,
		Status:      normalizePublicVideoStatus(dResp.Status),
		Progress:    dResp.Progress,
		SetProgress: true,
		CreatedAt:   dResp.CreatedAt,
		CompletedAt: dResp.CompletedAt,
		Duration:    request.Duration,
		Resolution:  request.Resolution,
		Ratio:       request.Ratio,
		Size:        dResp.Size,
	})
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "normalize_video_response_failed", http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", publicBody)
	return upstreamID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)

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
		// Url intentionally left empty — the caller constructs the proxy URL using the public task ID
	case "failed", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if resTask.Error != nil {
			taskResult.Reason = resTask.Error.Message
		} else {
			taskResult.Reason = "task failed"
		}
	default:
	}
	taskResult.Duration = responseDuration(resTask.Duration, resTask.Seconds)
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
	ID          string
	Model       string
	Status      string
	Progress    int
	SetProgress bool
	CreatedAt   int64
	CompletedAt int64
	Duration    int
	Resolution  string
	Ratio       string
	Size        string
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

func normalizePublicVideoResponse(data []byte, values publicVideoResponseValues) ([]byte, error) {
	if !gjson.ParseBytes(data).IsObject() {
		return nil, fmt.Errorf("video response must be a JSON object")
	}

	type fieldUpdate struct {
		path    string
		value   any
		enabled bool
	}
	updates := []fieldUpdate{
		{path: "id", value: values.ID, enabled: values.ID != ""},
		{path: "task_id", value: values.ID, enabled: values.ID != ""},
		{path: "object", value: "video", enabled: true},
		{path: "model", value: values.Model, enabled: values.Model != ""},
		{path: "status", value: values.Status, enabled: values.Status != ""},
		{path: "progress", value: values.Progress, enabled: values.SetProgress},
		{path: "created_at", value: values.CreatedAt, enabled: values.CreatedAt > 0},
		{path: "completed_at", value: values.CompletedAt, enabled: values.CompletedAt > 0},
		{path: "duration", value: values.Duration, enabled: values.Duration > 0},
		{path: "seconds", value: strconv.Itoa(values.Duration), enabled: values.Duration > 0},
		{path: "resolution", value: strings.ToLower(strings.TrimSpace(values.Resolution)), enabled: strings.TrimSpace(values.Resolution) != ""},
		{path: "ratio", value: strings.TrimSpace(values.Ratio), enabled: strings.TrimSpace(values.Ratio) != ""},
		{path: "size", value: strings.TrimSpace(values.Size), enabled: strings.TrimSpace(values.Size) != ""},
	}

	result := data
	for _, update := range updates {
		if !update.enabled {
			continue
		}
		updated, err := sjson.SetBytes(result, update.path, update.value)
		if err != nil {
			return nil, errors.Wrapf(err, "set public video response field %s failed", update.path)
		}
		result = updated
	}
	return result, nil
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
	if videoProtocolUsesConfiguredDuration(protocol) {
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
	duration := responseDuration(upstream.Duration, upstream.Seconds)
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
	createdAt := upstream.CreatedAt
	if task.SubmitTime > 0 {
		createdAt = task.SubmitTime
	}
	completedAt := upstream.CompletedAt
	if task.FinishTime > 0 {
		completedAt = task.FinishTime
	}

	data, err := normalizePublicVideoResponse(task.Data, publicVideoResponseValues{
		ID:          task.TaskID,
		Model:       publicModel,
		Status:      status,
		Progress:    progress,
		SetProgress: true,
		CreatedAt:   createdAt,
		CompletedAt: completedAt,
		Duration:    duration,
		Resolution:  resolution,
		Ratio:       ratio,
		Size:        size,
	})
	if err != nil {
		return nil, err
	}
	if task.Status == model.TaskStatusSuccess {
		proxyURL := taskcommon.BuildProxyURL(task.TaskID)
		for _, path := range []string{"url", "video_url", "metadata.url"} {
			data, err = sjson.SetBytes(data, path, proxyURL)
			if err != nil {
				return nil, errors.Wrapf(err, "set %s failed", path)
			}
		}
		for _, path := range []string{
			"metadata.content_url",
			"metadata.local_url",
			"metadata.video_url",
			"metadata.final_video_url",
		} {
			if !gjson.GetBytes(data, path).Exists() {
				continue
			}
			data, err = sjson.SetBytes(data, path, proxyURL)
			if err != nil {
				return nil, errors.Wrapf(err, "set %s failed", path)
			}
		}
		data, err = sjson.DeleteBytes(data, "metadata.origin_video_url")
		if err != nil {
			return nil, errors.Wrap(err, "delete metadata.origin_video_url failed")
		}
	} else {
		for _, path := range []string{
			"url",
			"video_url",
			"metadata.url",
			"metadata.content_url",
			"metadata.local_url",
			"metadata.video_url",
			"metadata.final_video_url",
			"metadata.origin_video_url",
		} {
			var err error
			data, err = sjson.DeleteBytes(data, path)
			if err != nil {
				return nil, errors.Wrapf(err, "delete %s failed", path)
			}
		}
	}

	return data, nil
}
