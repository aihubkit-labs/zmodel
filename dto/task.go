package dto

import (
	"encoding/json"
)

type TaskError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
	StatusCode int    `json:"-"`
	LocalError bool   `json:"-"`
	Error      error  `json:"-"`
}

type VideoParameterErrorData struct {
	Parameter         string   `json:"parameter"`
	Received          any      `json:"received,omitempty"`
	AllowedValues     []any    `json:"allowed_values,omitempty"`
	Minimum           *int64   `json:"minimum,omitempty"`
	Maximum           *int64   `json:"maximum,omitempty"`
	Required          *bool    `json:"required,omitempty"`
	RelatedParameters []string `json:"related_parameters,omitempty"`
}

type TaskData interface {
	SunoDataResponse | []SunoDataResponse | string | any
}

const TaskSuccessCode = "success"

type TaskResponse[T TaskData] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type TaskHTTPMessage struct {
	Method        string            `json:"method,omitempty"`
	URL           string            `json:"url,omitempty"`
	Protocol      string            `json:"protocol,omitempty"`
	Status        string            `json:"status,omitempty"`
	StatusCode    int               `json:"status_code,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          string            `json:"body,omitempty"`
	BodyTruncated bool              `json:"body_truncated,omitempty"`
	Error         string            `json:"error,omitempty"`
}

type TaskUpstreamHTTPTrace struct {
	PreparationRequest  *TaskHTTPMessage `json:"preparation_request,omitempty"`
	PreparationResponse *TaskHTTPMessage `json:"preparation_response,omitempty"`
	SubmitRequest       *TaskHTTPMessage `json:"submit_request,omitempty"`
	SubmitResponse      *TaskHTTPMessage `json:"submit_response,omitempty"`
	PollRequest         *TaskHTTPMessage `json:"poll_request,omitempty"`
	PollResponse        *TaskHTTPMessage `json:"poll_response,omitempty"`
}

func (t *TaskResponse[T]) IsSuccess() bool {
	return t.Code == TaskSuccessCode
}

type TaskDto struct {
	ID                    int64                  `json:"id"`
	CreatedAt             int64                  `json:"created_at"`
	UpdatedAt             int64                  `json:"updated_at"`
	TaskID                string                 `json:"task_id"`
	Platform              string                 `json:"platform"`
	PlatformName          string                 `json:"platform_name,omitempty"`
	UserId                int                    `json:"user_id"`
	Group                 string                 `json:"group"`
	ChannelId             int                    `json:"channel_id"`
	ChannelName           string                 `json:"channel_name,omitempty"`
	Quota                 int                    `json:"quota"`
	Action                string                 `json:"action"`
	Status                string                 `json:"status"`
	FailReason            string                 `json:"fail_reason"`
	ResultURL             string                 `json:"result_url,omitempty"` // 任务结果 URL（视频地址等）
	SubmitTime            int64                  `json:"submit_time"`
	StartTime             int64                  `json:"start_time"`
	FinishTime            int64                  `json:"finish_time"`
	Progress              string                 `json:"progress"`
	Properties            any                    `json:"properties"`
	Username              string                 `json:"username,omitempty"`
	Data                  json.RawMessage        `json:"data"`
	VideoS3StorageEnabled bool                   `json:"video_s3_storage_enabled,omitempty"`
	VideoStorageStatus    string                 `json:"video_storage_status,omitempty"`
	VideoStorageError     string                 `json:"video_storage_error,omitempty"`
	UpstreamHTTPTrace     *TaskUpstreamHTTPTrace `json:"upstream_http_trace,omitempty"`
}

type FetchReq struct {
	IDs []string `json:"ids"`
}
