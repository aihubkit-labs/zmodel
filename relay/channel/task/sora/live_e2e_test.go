package sora

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const liveVideoResponseLimit = 1 << 20

type liveVideoE2EConfig struct {
	name              string
	baseURL           string
	publicBaseURL     string
	apiKey            string
	model             string
	prompt            string
	duration          int
	ratio             string
	resolution        string
	pollInterval      time.Duration
	timeout           time.Duration
	expectZModelProxy bool
}

type liveVideoTaskResponse struct {
	ID       string `json:"id"`
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	VideoURL string `json:"video_url"`
	Metadata struct {
		URL string `json:"url"`
	} `json:"metadata"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

type liveVideoModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type liveVideoProtocolServer struct {
	URL string
}

func TestRunLiveVideoE2EAgainstProtocolServer(t *testing.T) {
	serverURL := ""
	requests := make(chan *http.Request, 4)
	server := newLiveVideoProtocolServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(r.Context())
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"videos-mini"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/videos":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"task_public","task_id":"task_public","status":"queued"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/videos/task_public":
			contentURL := serverURL + "/v1/videos/task_public/content"
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"id":"task_public","task_id":"task_public","status":"completed","url":%q,"video_url":%q,"metadata":{"url":%q}}`, contentURL, contentURL, contentURL)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/videos/task_public/content":
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Content-Range", "bytes 0-3/4")
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("test"))
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL = server.URL

	runLiveVideoE2E(t, liveVideoE2EConfig{
		name:              "protocol-test",
		baseURL:           server.URL,
		publicBaseURL:     server.URL,
		apiKey:            "test-zmodel-key",
		model:             "videos-mini",
		prompt:            "test prompt",
		duration:          5,
		ratio:             "16:9",
		resolution:        "720p",
		pollInterval:      time.Millisecond,
		timeout:           5 * time.Second,
		expectZModelProxy: true,
	})

	for range 4 {
		request := <-requests
		assert.Equal(t, "Bearer test-zmodel-key", request.Header.Get("Authorization"))
		if request.URL.Path == "/v1/videos/task_public/content" {
			assert.Equal(t, "bytes=0-1023", request.Header.Get("Range"))
		}
	}
}

func TestLiveVideoContentURLSupportsUpstreamResponseVariants(t *testing.T) {
	tests := []struct {
		name     string
		response liveVideoTaskResponse
		expected string
	}{
		{
			name:     "url",
			response: liveVideoTaskResponse{URL: "https://example.com/url"},
			expected: "https://example.com/url",
		},
		{
			name:     "video url",
			response: liveVideoTaskResponse{VideoURL: "https://example.com/video-url"},
			expected: "https://example.com/video-url",
		},
		{
			name: "metadata url",
			response: func() liveVideoTaskResponse {
				response := liveVideoTaskResponse{}
				response.Metadata.URL = "https://example.com/metadata-url"
				return response
			}(),
			expected: "https://example.com/metadata-url",
		},
		{
			name: "url takes precedence",
			response: func() liveVideoTaskResponse {
				response := liveVideoTaskResponse{
					URL:      "https://example.com/url",
					VideoURL: "https://example.com/video-url",
				}
				response.Metadata.URL = "https://example.com/metadata-url"
				return response
			}(),
			expected: "https://example.com/url",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, liveVideoContentURL(test.response))
		})
	}
}

func TestLiveVideoDiagnosticRedactsAPIKey(t *testing.T) {
	apiKey := "test-secret-key"
	diagnostic := liveVideoDiagnostic([]byte(`{"error":"credential test-secret-key rejected"}`), apiKey)

	assert.NotContains(t, diagnostic, apiKey)
	assert.Contains(t, diagnostic, "[REDACTED]")
}

func TestLiveOpenAIVideoE2E(t *testing.T) {
	if !liveVideoE2EEnabled(t) {
		t.Skip("set OPENAI_VIDEO_E2E_ENABLED=true to run the billable live video test")
	}

	configs := loadLiveVideoE2EConfigs(t)
	for _, config := range configs {
		config := config
		t.Run(config.name, func(t *testing.T) {
			runLiveVideoE2E(t, config)
		})
	}
}

func liveVideoE2EEnabled(t *testing.T) bool {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv("OPENAI_VIDEO_E2E_ENABLED"))
	if raw == "" {
		return false
	}
	enabled, err := strconv.ParseBool(raw)
	require.NoError(t, err, "OPENAI_VIDEO_E2E_ENABLED must be a boolean")
	return enabled
}

func loadLiveVideoE2EConfigs(t *testing.T) []liveVideoE2EConfig {
	t.Helper()

	target := strings.ToLower(strings.TrimSpace(os.Getenv("OPENAI_VIDEO_E2E_TARGET")))
	if target == "" {
		target = "zmodel"
	}
	require.Contains(t, []string{"zmodel", "upstream", "both"}, target,
		"OPENAI_VIDEO_E2E_TARGET must be zmodel, upstream, or both")

	modelName := liveVideoEnvOrDefault("OPENAI_VIDEO_E2E_MODEL", "videos-mini")
	prompt := liveVideoEnvOrDefault("OPENAI_VIDEO_E2E_PROMPT", "A paper airplane glides through a quiet sunlit room")
	duration := liveVideoPositiveIntEnv(t, "OPENAI_VIDEO_E2E_DURATION", 5)
	pollInterval := liveVideoPositiveDurationEnv(t, "OPENAI_VIDEO_E2E_POLL_INTERVAL", 5*time.Second)
	timeout := liveVideoPositiveDurationEnv(t, "OPENAI_VIDEO_E2E_TIMEOUT", 15*time.Minute)
	commonConfig := liveVideoE2EConfig{
		model:        modelName,
		prompt:       prompt,
		duration:     duration,
		ratio:        liveVideoEnvOrDefault("OPENAI_VIDEO_E2E_RATIO", "16:9"),
		resolution:   liveVideoEnvOrDefault("OPENAI_VIDEO_E2E_RESOLUTION", "720p"),
		pollInterval: pollInterval,
		timeout:      timeout,
	}

	configs := make([]liveVideoE2EConfig, 0, 2)
	if target == "zmodel" || target == "both" {
		baseURL := liveVideoRequiredEnv(t, "ZMODEL_BASE_URL")
		config := commonConfig
		config.name = "zmodel"
		config.baseURL = baseURL
		config.publicBaseURL = liveVideoEnvOrDefault("ZMODEL_PUBLIC_BASE_URL", baseURL)
		config.apiKey = liveVideoRequiredEnv(t, "ZMODEL_API_KEY")
		config.expectZModelProxy = true
		configs = append(configs, config)
	}
	if target == "upstream" || target == "both" {
		config := commonConfig
		config.name = "upstream"
		config.baseURL = liveVideoEnvOrDefault("FRIMODEL_BASE_URL", "https://api.frimodel.com")
		config.publicBaseURL = config.baseURL
		config.apiKey = liveVideoRequiredEnv(t, "FRIMODEL_API_KEY")
		configs = append(configs, config)
	}
	return configs
}

func runLiveVideoE2E(t *testing.T, config liveVideoE2EConfig) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), config.timeout)
	defer cancel()
	client := &http.Client{Timeout: 60 * time.Second}

	modelsURL := strings.TrimRight(config.baseURL, "/") + "/v1/models"
	modelsResponse := liveVideoModelsResponse{}
	liveVideoJSONRequest(t, ctx, client, http.MethodGet, modelsURL, config.apiKey, nil, &modelsResponse)
	modelFound := false
	for _, item := range modelsResponse.Data {
		if item.ID == config.model {
			modelFound = true
			break
		}
	}
	require.Truef(t, modelFound, "configured model %q was not returned by %s", config.model, modelsURL)

	submitBody := map[string]any{
		"model":      config.model,
		"prompt":     config.prompt,
		"duration":   config.duration,
		"ratio":      config.ratio,
		"resolution": config.resolution,
	}
	submitResponse := liveVideoTaskResponse{}
	submitURL := strings.TrimRight(config.baseURL, "/") + "/v1/videos"
	liveVideoJSONRequest(t, ctx, client, http.MethodPost, submitURL, config.apiKey, submitBody, &submitResponse)

	publicTaskID := submitResponse.ID
	if publicTaskID == "" {
		publicTaskID = submitResponse.TaskID
	}
	require.NotEmpty(t, publicTaskID, "video submit response did not contain id or task_id")
	if config.expectZModelProxy {
		assert.Equal(t, publicTaskID, submitResponse.ID)
		assert.Equal(t, publicTaskID, submitResponse.TaskID)
		assert.True(t, strings.HasPrefix(publicTaskID, "task_"), "zmodel task ID must use the public task_ prefix")
	}

	queryURL := strings.TrimRight(config.baseURL, "/") + "/v1/videos/" + url.PathEscape(publicTaskID)
	var completed liveVideoTaskResponse
	for {
		if err := ctx.Err(); err != nil {
			t.Fatalf("video task %s did not complete within %s", publicTaskID, config.timeout)
		}

		current := liveVideoTaskResponse{}
		liveVideoJSONRequest(t, ctx, client, http.MethodGet, queryURL, config.apiKey, nil, &current)
		if config.expectZModelProxy {
			assert.Equal(t, publicTaskID, current.ID)
			assert.Equal(t, publicTaskID, current.TaskID)
		}

		switch current.Status {
		case dto.VideoStatusCompleted:
			completed = current
		case dto.VideoStatusFailed, "cancelled":
			message := ""
			if current.Error != nil {
				message = liveVideoDiagnostic([]byte(current.Error.Message), config.apiKey)
			}
			t.Fatalf("video task %s ended with status %q: %s", publicTaskID, current.Status, message)
		case dto.VideoStatusQueued, dto.VideoStatusInProgress, "pending", "processing", "submitted":
			if config.expectZModelProxy {
				assert.Empty(t, current.URL)
				assert.Empty(t, current.VideoURL)
				assert.Empty(t, current.Metadata.URL)
			}
			select {
			case <-ctx.Done():
				t.Fatalf("video task %s did not complete within %s", publicTaskID, config.timeout)
			case <-time.After(config.pollInterval):
				continue
			}
		default:
			t.Fatalf("video task %s returned unsupported status %q", publicTaskID, current.Status)
		}
		break
	}

	contentURL := liveVideoContentURL(completed)
	require.NotEmpty(t, contentURL, "completed video response did not contain a content URL")
	if config.expectZModelProxy {
		expectedContentURL := strings.TrimRight(config.publicBaseURL, "/") + "/v1/videos/" + url.PathEscape(publicTaskID) + "/content"
		assert.Equal(t, expectedContentURL, completed.URL)
		assert.Equal(t, expectedContentURL, completed.VideoURL)
		assert.Equal(t, expectedContentURL, completed.Metadata.URL)
		assert.Equal(t, expectedContentURL, contentURL)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, contentURL, nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+config.apiKey)
	request.Header.Set("Range", "bytes=0-1023")
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	require.NoError(t, err)

	require.Equal(t, http.StatusPartialContent, response.StatusCode,
		"content endpoint must honor Range requests; response body: %s", liveVideoDiagnostic(content, config.apiKey))
	assert.NotEmpty(t, response.Header.Get("Content-Range"))
	assert.Equal(t, "bytes", strings.ToLower(response.Header.Get("Accept-Ranges")))
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	assert.True(t, strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "application/octet-stream"),
		"unexpected video content type %q", contentType)

	require.NotEmpty(t, content, "content endpoint returned an empty byte range")
}

func liveVideoJSONRequest(t *testing.T, ctx context.Context, client *http.Client, method, requestURL, apiKey string, body any, target any) {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		encoded, err := common.Marshal(body)
		require.NoError(t, err)
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, liveVideoResponseLimit))
	require.NoError(t, err)
	diagnostic := liveVideoDiagnostic(responseBody, apiKey)
	require.GreaterOrEqual(t, response.StatusCode, http.StatusOK, "request to %s failed: %s", requestURL, diagnostic)
	require.Less(t, response.StatusCode, http.StatusMultipleChoices, "request to %s failed: %s", requestURL, diagnostic)
	require.NoError(t, common.Unmarshal(responseBody, target), "invalid JSON response from %s: %s", requestURL, diagnostic)
}

func liveVideoContentURL(response liveVideoTaskResponse) string {
	if response.URL != "" {
		return response.URL
	}
	if response.VideoURL != "" {
		return response.VideoURL
	}
	return response.Metadata.URL
}

func liveVideoDiagnostic(body []byte, apiKey string) string {
	diagnostic := string(body)
	if apiKey != "" {
		diagnostic = strings.ReplaceAll(diagnostic, apiKey, "[REDACTED]")
	}
	return diagnostic
}

func liveVideoRequiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	require.NotEmptyf(t, value, "%s is required for the selected live E2E target", name)
	return value
}

func liveVideoEnvOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func liveVideoPositiveIntEnv(t *testing.T, name string, fallback int) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	require.NoErrorf(t, err, "%s must be an integer", name)
	require.Positivef(t, value, "%s must be positive", name)
	return value
}

func liveVideoPositiveDurationEnv(t *testing.T, name string, fallback time.Duration) time.Duration {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	require.NoErrorf(t, err, "%s must be a Go duration such as 5s or 15m", name)
	require.Positivef(t, value, "%s must be positive", name)
	return value
}

func newLiveVideoProtocolServer(t *testing.T, handler http.Handler) *liveVideoProtocolServer {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	httpServer := &http.Server{Handler: handler}
	go func() {
		_ = httpServer.Serve(listener)
	}()
	t.Cleanup(func() {
		require.NoError(t, httpServer.Close())
	})

	return &liveVideoProtocolServer{
		URL: "http://" + listener.Addr().String(),
	}
}
