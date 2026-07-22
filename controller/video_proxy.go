package controller

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// videoProxyError returns a standardized OpenAI-style error response.
func videoProxyError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

func VideoProxy(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "task_id is required")
		return
	}

	userID := c.GetInt("id")
	task, exists, err := model.GetByTaskId(userID, taskID)
	if err == nil && !exists && model.IsAdmin(userID) {
		task, exists, err = model.GetByOnlyTaskId(taskID)
	}
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to query task %s: %s", taskID, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to query task")
		return
	}
	if !exists || task == nil {
		videoProxyError(c, http.StatusNotFound, "invalid_request_error", "Task not found")
		return
	}

	if task.Status != model.TaskStatusSuccess {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error",
			fmt.Sprintf("Task is not completed yet, current status: %s", task.Status))
		return
	}

	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to get channel for task %s: %s", taskID, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to retrieve channel information")
		return
	}
	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	channelSetting := channel.GetSetting()
	var videoURL string
	var videoAPIKey string
	proxy := channelSetting.Proxy
	taskClient := service.GetSSRFProtectedHTTPClient()
	videoClient := service.GetVideoContentHTTPClient()
	if proxy != "" {
		// 渠道代理路径的连接由代理侧建立，无法做拨号时逐 IP 校验，
		// 因此后面对 videoURL 保留请求前的一次性 SSRF 校验。
		taskClient, err = service.GetHttpClientWithProxy(proxy)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create proxy client for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy client")
			return
		}
		videoClient = taskClient
	}
	switch channel.Type {
	case constant.ChannelTypeGemini:
		apiKey := task.PrivateData.Key
		if apiKey == "" {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Missing stored API key for Gemini task %s", taskID))
			videoProxyError(c, http.StatusInternalServerError, "server_error", "API key not stored for task")
			return
		}
		videoAPIKey = apiKey
		videoURL, err = getGeminiVideoURL(channel, task, apiKey)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve Gemini video URL for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to resolve Gemini video URL")
			return
		}
	case constant.ChannelTypeVertexAi:
		videoURL, err = getVertexVideoURL(channel, task)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve Vertex video URL for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to resolve Vertex video URL")
			return
		}
	case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
		upstreamKey := task.PrivateData.Key
		if upstreamKey == "" {
			upstreamKey = channel.Key
		}
		videoURL, err = fetchOpenAIVideoTaskURL(c, taskClient, baseURL, task.GetUpstreamTaskID(), upstreamKey, proxy)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve OpenAI video URL for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusBadGateway, "server_error", err.Error())
			return
		}
	default:
		// Video URL is stored in PrivateData.ResultURL (fallback to FailReason for old data)
		videoURL = task.GetResultURL()
	}

	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Video URL is empty for task %s", taskID))
		videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		return
	}

	if strings.HasPrefix(videoURL, "data:") {
		if err := writeVideoDataURL(c, videoURL); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to decode video data URL for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		}
		return
	}

	parsedVideoURL, err := url.Parse(videoURL)
	if err != nil || parsedVideoURL.Host == "" || (parsedVideoURL.Scheme != "http" && parsedVideoURL.Scheme != "https") {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Invalid video URL for task %s", taskID))
		videoProxyError(c, http.StatusBadGateway, "server_error", "Task detail response contains an invalid url")
		return
	}
	if !channelSetting.VideoContentProxyEnabled {
		if parsedVideoURL.Scheme != "https" {
			logger.LogError(c.Request.Context(), fmt.Sprintf("HTTP video URL requires proxy for task %s", taskID))
			videoProxyError(c, http.StatusBadGateway, "server_error", "Upstream returned an HTTP video URL; enable video content proxy for this channel")
			return
		}
		if err := validateVideoRedirectURL(videoURL); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Video URL blocked for task %s: %v", taskID, err))
			videoProxyError(c, http.StatusForbidden, "server_error", fmt.Sprintf("request blocked: %v", err))
			return
		}
		videoURL, err = resolveFinalVideoURL(c, videoClient, videoURL, videoAPIKey)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve final video URL for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to resolve final video URL")
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, videoURL)
		return
	}

	if err := validateVideoFetchURL(videoURL, proxy); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Video URL blocked for task %s: %v", taskID, err))
		videoProxyError(c, http.StatusForbidden, "server_error", fmt.Sprintf("request blocked: %v", err))
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, videoURL, nil)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create video request for %s: %s", videoURL, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy request")
		return
	}
	if videoAPIKey != "" {
		req.Header.Set("x-goog-api-key", videoAPIKey)
	}
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	if ifRangeHeader := c.GetHeader("If-Range"); ifRangeHeader != "" {
		req.Header.Set("If-Range", ifRangeHeader)
	}

	resp, err := videoClient.Do(req)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to fetch video from %s: %s", videoURL, err.Error()))
		videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Upstream returned status %d for %s", resp.StatusCode, videoURL))
		videoProxyError(c, http.StatusBadGateway, "server_error",
			fmt.Sprintf("Upstream service returned status %d", resp.StatusCode))
		return
	}

	responseHeaders := []string{
		"Content-Type",
		"Content-Length",
		"Content-Range",
		"Accept-Ranges",
		"Content-Disposition",
		"ETag",
		"Last-Modified",
	}
	for _, key := range responseHeaders {
		values := resp.Header.Values(key)
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Writer.Header().Set("Cache-Control", "private, max-age=86400")
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to stream video content: %s", err.Error()))
	}
}

func fetchOpenAIVideoTaskURL(c *gin.Context, client *http.Client, baseURL, upstreamTaskID, key, proxy string) (string, error) {
	taskDetailURL := fmt.Sprintf("%s/v1/videos/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(upstreamTaskID))
	if err := validateVideoFetchURL(taskDetailURL, proxy); err != nil {
		return "", fmt.Errorf("task detail request blocked: %w", err)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, taskDetailURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create task detail request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch task details: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("upstream task detail returned status %d", resp.StatusCode)
	}

	var taskDetail struct {
		URL      string `json:"url"`
		VideoURL string `json:"video_url"`
		Video    struct {
			URL string `json:"url"`
		} `json:"video"`
		Metadata struct {
			URL           string `json:"url"`
			ContentURL    string `json:"content_url"`
			LocalURL      string `json:"local_url"`
			VideoURL      string `json:"video_url"`
			FinalVideoURL string `json:"final_video_url"`
		} `json:"metadata"`
	}
	if err := common.DecodeJson(resp.Body, &taskDetail); err != nil {
		return "", fmt.Errorf("failed to decode task detail response: %w", err)
	}
	videoURL := strings.TrimSpace(taskDetail.URL)
	if videoURL == "" {
		videoURL = strings.TrimSpace(taskDetail.Video.URL)
	}
	if videoURL == "" {
		videoURL = strings.TrimSpace(taskDetail.VideoURL)
	}
	if videoURL == "" {
		videoURL = strings.TrimSpace(taskDetail.Metadata.URL)
	}
	if videoURL == "" {
		videoURL = strings.TrimSpace(taskDetail.Metadata.ContentURL)
	}
	if videoURL == "" {
		videoURL = strings.TrimSpace(taskDetail.Metadata.LocalURL)
	}
	if videoURL == "" {
		videoURL = strings.TrimSpace(taskDetail.Metadata.VideoURL)
	}
	if videoURL == "" {
		videoURL = strings.TrimSpace(taskDetail.Metadata.FinalVideoURL)
	}
	if videoURL == "" {
		return "", fmt.Errorf("Task detail response does not contain url")
	}
	return videoURL, nil
}

func validateVideoFetchURL(rawURL, proxy string) error {
	if proxy == "" {
		return service.ValidateSSRFProtectedFetchURL(rawURL)
	}
	fetchSetting := system_setting.GetFetchSetting()
	return common.ValidateURLWithFetchSetting(rawURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain)
}

func validateVideoRedirectURL(rawURL string) error {
	fetchSetting := system_setting.GetFetchSetting()
	return common.ValidateURLWithFetchSetting(rawURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, nil, fetchSetting.ApplyIPFilterForDomain)
}

func resolveFinalVideoURL(c *gin.Context, client *http.Client, videoURL, videoAPIKey string) (string, error) {
	probeClient := &http.Client{
		Transport: client.Transport,
		Timeout:   client.Timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	currentURL := videoURL
	for range 10 {
		if err := validateVideoRedirectURL(currentURL); err != nil {
			return "", fmt.Errorf("video redirect blocked: %w", err)
		}
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, currentURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create video probe request: %w", err)
		}
		req.Header.Set("Range", "bytes=0-0")
		if videoAPIKey != "" {
			req.Header.Set("x-goog-api-key", videoAPIKey)
		}
		resp, err := probeClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to probe video URL: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
			return currentURL, nil
		}
		if !isVideoRedirectStatus(resp.StatusCode) {
			return "", fmt.Errorf("video URL returned status %d", resp.StatusCode)
		}

		location := strings.TrimSpace(resp.Header.Get("Location"))
		if location == "" {
			return "", fmt.Errorf("video redirect missing Location header")
		}
		base, err := url.Parse(currentURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse video URL: %w", err)
		}
		target, err := base.Parse(location)
		if err != nil {
			return "", fmt.Errorf("failed to parse video redirect URL: %w", err)
		}
		currentURL = target.String()
	}
	return "", fmt.Errorf("stopped after 10 video redirects")
}

func isVideoRedirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func writeVideoDataURL(c *gin.Context, dataURL string) error {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid data url")
	}

	header := parts[0]
	payload := parts[1]
	if !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return fmt.Errorf("unsupported data url")
	}

	mimeType := strings.TrimPrefix(header, "data:")
	mimeType = strings.TrimSuffix(mimeType, ";base64")
	if mimeType == "" {
		mimeType = "video/mp4"
	}

	videoBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		videoBytes, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return err
		}
	}

	c.Writer.Header().Set("Content-Type", mimeType)
	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	c.Writer.WriteHeader(http.StatusOK)
	_, err = c.Writer.Write(videoBytes)
	return err
}
