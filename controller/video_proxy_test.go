package controller

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type videoProxyUpstreamRequest struct {
	Path          string
	Authorization string
	Range         string
	IfRange       string
}

type videoProxyTestCase struct {
	storedKey             string
	channelKey            string
	expectedKey           string
	upstreamStatus        int
	expectedStatus        int
	forwardRange          bool
	expectedResponseBody  string
	expectedContentLength string
}

func setupVideoProxyTest(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalFetchSetting := *system_setting.GetFetchSetting()

	common.MemoryCacheEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	fetchSetting := system_setting.GetFetchSetting()
	fetchSetting.EnableSSRFProtection = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.Channel{}))
	require.NoError(t, db.Exec("CREATE TABLE IF NOT EXISTS users (id integer primary key, role integer, deleted_at datetime)").Error)

	service.InitHttpClient()
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		*system_setting.GetFetchSetting() = originalFetchSetting
		service.InitHttpClient()
	})

	return db
}

func TestVideoProxyUsesStoredTaskKeyAndForwardsRange(t *testing.T) {
	testVideoProxyDownload(t, videoProxyTestCase{
		storedKey:             "stored-task-key",
		channelKey:            "current-channel-key",
		expectedKey:           "stored-task-key",
		upstreamStatus:        http.StatusPartialContent,
		expectedStatus:        http.StatusPartialContent,
		forwardRange:          true,
		expectedResponseBody:  "test",
		expectedContentLength: "4",
	})
}

func TestVideoProxyFallsBackToCurrentChannelKeyForHistoricalTask(t *testing.T) {
	testVideoProxyDownload(t, videoProxyTestCase{
		channelKey:            "current-channel-key",
		expectedKey:           "current-channel-key",
		upstreamStatus:        http.StatusPartialContent,
		expectedStatus:        http.StatusPartialContent,
		forwardRange:          true,
		expectedResponseBody:  "test",
		expectedContentLength: "4",
	})
}

func TestVideoProxyStreamsFullUpstreamResponse(t *testing.T) {
	testVideoProxyDownload(t, videoProxyTestCase{
		storedKey:             "stored-task-key",
		channelKey:            "current-channel-key",
		expectedKey:           "stored-task-key",
		upstreamStatus:        http.StatusOK,
		expectedStatus:        http.StatusOK,
		expectedResponseBody:  "test",
		expectedContentLength: "4",
	})
}

func TestVideoProxyAllowsAdminToPreviewAnotherUsersTask(t *testing.T) {
	db := setupVideoProxyTest(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/video.mp4" {
			w.Header().Set("Content-Type", "video/mp4")
			_, _ = w.Write([]byte("test"))
			return
		}
		_, _ = fmt.Fprintf(w, `{"status":"completed","url":%q}`, "http://"+r.Host+"/video.mp4")
	}))
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	setting := `{"video_content_proxy_enabled":true}`
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     "channel-key",
		Name:    "OpenAI video",
		BaseURL: &baseURL,
		Setting: &setting,
	}
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Exec("INSERT INTO users (id, role) VALUES (?, ?)", 501, common.RoleAdminUser).Error)

	task := &model.Task{
		TaskID:    "task_owned_by_another_user",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream_task_id",
		},
		Data: []byte(`{"status":"completed"}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_owned_by_another_user/content", nil)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", 501)

	VideoProxy(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "test", recorder.Body.String())
}

func TestVideoProxyDoesNotAllowUserToPreviewAnotherUsersTask(t *testing.T) {
	db := setupVideoProxyTest(t)

	task := &model.Task{
		TaskID: "task_owned_by_another_user",
		UserId: 401,
		Status: model.TaskStatusSuccess,
		Data:   []byte(`{"status":"completed"}`),
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, db.Exec("INSERT INTO users (id, role) VALUES (?, ?)", 502, common.RoleCommonUser).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_owned_by_another_user/content", nil)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", 502)

	VideoProxy(context)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Task not found")
}

func TestVideoProxyPreservesGeminiVideoAuthentication(t *testing.T) {
	db := setupVideoProxyTest(t)

	var apiKeyHeader string
	var apiKeyQuery string
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKeyHeader = r.Header.Get("x-goog-api-key")
		apiKeyQuery = r.URL.Query().Get("key")
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("test"))
	}))
	t.Cleanup(videoServer.Close)

	setting := `{"video_content_proxy_enabled":true}`
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeGemini,
		Name:    "Gemini video",
		Setting: &setting,
	}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_gemini_video",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			Key: "stored-gemini-key",
		},
		Data: []byte(fmt.Sprintf(`{"uri":%q}`, videoServer.URL+"/video.mp4")),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_gemini_video/content", nil)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", task.UserId)

	VideoProxy(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "stored-gemini-key", apiKeyHeader)
	assert.Equal(t, "stored-gemini-key", apiKeyQuery)
}

func TestVideoProxyMapsUpstreamFailureToBadGateway(t *testing.T) {
	testVideoProxyDownload(t, videoProxyTestCase{
		storedKey:      "stored-task-key",
		channelKey:     "current-channel-key",
		expectedKey:    "stored-task-key",
		upstreamStatus: http.StatusUnauthorized,
		expectedStatus: http.StatusBadGateway,
	})
}

func TestVideoProxyRedirectsToFreshTaskDetailURL(t *testing.T) {
	db := setupVideoProxyTest(t)
	originalTLSInsecureSkipVerify := common.TLSInsecureSkipVerify
	common.TLSInsecureSkipVerify = true
	service.InitHttpClient()
	t.Cleanup(func() {
		common.TLSInsecureSkipVerify = originalTLSInsecureSkipVerify
		service.InitHttpClient()
	})

	videoServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Range", "bytes 0-0/4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("t"))
	}))
	t.Cleanup(videoServer.Close)

	var upstreamPath string
	var upstreamAuthorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		upstreamAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"completed","url":%q}`, videoServer.URL+"/video.mp4")
	}))
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     "current-channel-key",
		Name:    "OpenAI video",
		BaseURL: &baseURL,
	}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_zmodel_public",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			Key:            "stored-task-key",
			UpstreamTaskID: "task_frimodel_upstream",
		},
		Data: []byte(`{"status":"completed","url":"https://expired.example/video.mp4"}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_zmodel_public/content", nil)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", task.UserId)

	VideoProxy(context)

	require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, videoServer.URL+"/video.mp4", recorder.Header().Get("Location"))
	assert.Equal(t, "/v1/videos/task_frimodel_upstream", upstreamPath)
	assert.Equal(t, "Bearer stored-task-key", upstreamAuthorization)
}

func TestVideoProxyRedirectsToFinalVideoURL(t *testing.T) {
	db := setupVideoProxyTest(t)
	originalTLSInsecureSkipVerify := common.TLSInsecureSkipVerify
	common.TLSInsecureSkipVerify = true
	service.InitHttpClient()
	t.Cleanup(func() {
		common.TLSInsecureSkipVerify = originalTLSInsecureSkipVerify
		service.InitHttpClient()
	})

	finalVideoServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Range", "bytes 0-0/4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("t"))
	}))
	t.Cleanup(finalVideoServer.Close)

	videoRedirectServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", finalVideoServer.URL+"/video.mp4?signature=fresh")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	t.Cleanup(videoRedirectServer.Close)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"completed","url":%q}`, videoRedirectServer.URL+"/content")
	}))
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     "channel-key",
		Name:    "OpenAI video",
		BaseURL: &baseURL,
	}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_video_redirect_chain",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "task_frimodel_upstream",
		},
		Data: []byte(`{"status":"completed"}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_video_redirect_chain/content", nil)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", task.UserId)

	VideoProxy(context)

	require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, finalVideoServer.URL+"/video.mp4?signature=fresh", recorder.Header().Get("Location"))
}

func TestVideoProxyRedirectAllowsPublicHTTPSCustomPort(t *testing.T) {
	db := setupVideoProxyTest(t)
	originalTLSInsecureSkipVerify := common.TLSInsecureSkipVerify
	common.TLSInsecureSkipVerify = true
	service.InitHttpClient()
	t.Cleanup(func() {
		common.TLSInsecureSkipVerify = originalTLSInsecureSkipVerify
		service.InitHttpClient()
	})

	videoServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Range", "bytes 0-0/4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("t"))
	}))
	t.Cleanup(videoServer.Close)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"completed","url":%q}`, videoServer.URL+"/video.mp4")
	}))
	t.Cleanup(upstream.Close)

	upstreamURL, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	_, upstreamPort, err := net.SplitHostPort(upstreamURL.Host)
	require.NoError(t, err)

	fetchSetting := system_setting.GetFetchSetting()
	fetchSetting.EnableSSRFProtection = true
	fetchSetting.AllowPrivateIp = true
	fetchSetting.DomainFilterMode = false
	fetchSetting.IpFilterMode = false
	fetchSetting.AllowedPorts = []string{upstreamPort}
	service.InitHttpClient()

	baseURL := upstream.URL
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     "channel-key",
		Name:    "OpenAI video",
		BaseURL: &baseURL,
	}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_custom_video_port",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "task_frimodel_upstream",
		},
		Data: []byte(`{"status":"completed"}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_custom_video_port/content", nil)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", task.UserId)

	VideoProxy(context)

	require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, videoServer.URL+"/video.mp4", recorder.Header().Get("Location"))
}

func TestValidateVideoRedirectURLRejectsPrivateAddress(t *testing.T) {
	setupVideoProxyTest(t)

	fetchSetting := system_setting.GetFetchSetting()
	fetchSetting.EnableSSRFProtection = true
	fetchSetting.AllowPrivateIp = false
	fetchSetting.DomainFilterMode = false
	fetchSetting.IpFilterMode = false

	err := validateVideoRedirectURL("https://127.0.0.1:19443/video.mp4")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "private IP address not allowed")
}

func TestVideoProxyStreamsFreshTaskDetailURLWhenEnabled(t *testing.T) {
	db := setupVideoProxyTest(t)

	var receivedRange string
	var receivedIfRange string
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRange = r.Header.Get("Range")
		receivedIfRange = r.Header.Get("If-Range")
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Range", "bytes 0-3/10")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", "4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("test"))
	}))
	t.Cleanup(videoServer.Close)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/videos/task_frimodel_upstream", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"completed","url":%q}`, videoServer.URL+"/video.mp4")
	}))
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	setting := `{"video_content_proxy_enabled":true}`
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     "channel-key",
		Name:    "OpenAI video",
		BaseURL: &baseURL,
		Setting: &setting,
	}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_zmodel_public",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "task_frimodel_upstream",
		},
		Data: []byte(`{"status":"completed"}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_zmodel_public/content", nil)
	context.Request.Header.Set("Range", "bytes=0-3")
	context.Request.Header.Set("If-Range", `"video-etag"`)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", task.UserId)

	VideoProxy(context)

	require.Equal(t, http.StatusPartialContent, recorder.Code)
	assert.Equal(t, "test", recorder.Body.String())
	assert.Equal(t, "bytes=0-3", receivedRange)
	assert.Equal(t, `"video-etag"`, receivedIfRange)
}

func TestVideoProxyRejectsHTTPTaskDetailURLWhenProxyDisabled(t *testing.T) {
	db := setupVideoProxyTest(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","url":"http://video.example/video.mp4"}`))
	}))
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     "channel-key",
		Name:    "OpenAI video",
		BaseURL: &baseURL,
	}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_zmodel_public",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "task_frimodel_upstream",
		},
		Data: []byte(`{"status":"completed"}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_zmodel_public/content", nil)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", task.UserId)

	VideoProxy(context)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Empty(t, recorder.Header().Get("Location"))
	assert.Contains(t, recorder.Body.String(), "enable video content proxy")
}

func TestVideoProxyRequiresTopLevelTaskDetailURL(t *testing.T) {
	db := setupVideoProxyTest(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","video_url":"https://video.example/video.mp4","metadata":{"url":"https://video.example/metadata.mp4"}}`))
	}))
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     "channel-key",
		Name:    "OpenAI video",
		BaseURL: &baseURL,
	}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_zmodel_public",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "task_frimodel_upstream",
		},
		Data: []byte(`{"status":"completed"}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_zmodel_public/content", nil)
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", task.UserId)

	VideoProxy(context)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Task detail response does not contain url")
}

func testVideoProxyDownload(t *testing.T, testCase videoProxyTestCase) {
	t.Helper()
	db := setupVideoProxyTest(t)

	videoRequest := make(chan videoProxyUpstreamRequest, 1)
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		videoRequest <- videoProxyUpstreamRequest{
			Path:    r.URL.Path,
			Range:   r.Header.Get("Range"),
			IfRange: r.Header.Get("If-Range"),
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Range", "bytes 0-3/10")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Disposition", `inline; filename="video.mp4"`)
		w.Header().Set("ETag", `"video-etag"`)
		w.Header().Set("Last-Modified", "Wed, 15 Jul 2026 10:00:00 GMT")
		w.Header().Set("X-Upstream-Internal", "must-not-leak")
		w.Header().Set("Content-Length", "4")
		w.WriteHeader(testCase.upstreamStatus)
		_, _ = w.Write([]byte("test"))
	}))
	t.Cleanup(videoServer.Close)

	taskDetailRequest := make(chan videoProxyUpstreamRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		taskDetailRequest <- videoProxyUpstreamRequest{
			Path:          r.URL.Path,
			Authorization: r.Header.Get("Authorization"),
		}
		if testCase.upstreamStatus == http.StatusUnauthorized {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = fmt.Fprintf(w, `{"status":"completed","url":%q}`, videoServer.URL+"/video.mp4")
	}))
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	setting := `{"video_content_proxy_enabled":true}`
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     testCase.channelKey,
		Name:    "FriModel OpenAI video",
		BaseURL: &baseURL,
		Setting: &setting,
	}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID:    "task_zmodel_public",
		UserId:    401,
		ChannelId: channel.Id,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			Key:            testCase.storedKey,
			UpstreamTaskID: "task_frimodel_upstream",
		},
		Data: []byte(`{"status":"completed"}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_zmodel_public/content", nil)
	if testCase.forwardRange {
		context.Request.Header.Set("Range", "bytes=0-3")
		context.Request.Header.Set("If-Range", `"video-etag"`)
	}
	context.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	context.Set("id", task.UserId)

	VideoProxy(context)

	require.Equal(t, testCase.expectedStatus, recorder.Code)
	if testCase.expectedStatus == http.StatusBadGateway {
		assert.Contains(t, recorder.Body.String(), "upstream task detail returned status 401")
		assert.Empty(t, recorder.Header().Get("Cache-Control"))
		assert.Empty(t, recorder.Header().Get("X-Upstream-Internal"))
	} else {
		assert.Equal(t, testCase.expectedResponseBody, recorder.Body.String())
		assert.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
		assert.Equal(t, testCase.expectedContentLength, recorder.Header().Get("Content-Length"))
		assert.Equal(t, "bytes 0-3/10", recorder.Header().Get("Content-Range"))
		assert.Equal(t, "bytes", recorder.Header().Get("Accept-Ranges"))
		assert.Equal(t, `inline; filename="video.mp4"`, recorder.Header().Get("Content-Disposition"))
		assert.Equal(t, `"video-etag"`, recorder.Header().Get("ETag"))
		assert.Equal(t, "Wed, 15 Jul 2026 10:00:00 GMT", recorder.Header().Get("Last-Modified"))
		assert.Equal(t, "private, max-age=86400", recorder.Header().Get("Cache-Control"))
		assert.Empty(t, recorder.Header().Get("X-Upstream-Internal"))
	}

	detailReceived := <-taskDetailRequest
	assert.Equal(t, "/v1/videos/task_frimodel_upstream", detailReceived.Path)
	assert.Equal(t, "Bearer "+testCase.expectedKey, detailReceived.Authorization)
	if testCase.expectedStatus != http.StatusBadGateway {
		videoReceived := <-videoRequest
		assert.Equal(t, "/video.mp4", videoReceived.Path)
		if testCase.forwardRange {
			assert.Equal(t, "bytes=0-3", videoReceived.Range)
			assert.Equal(t, `"video-etag"`, videoReceived.IfRange)
		} else {
			assert.Empty(t, videoReceived.Range)
			assert.Empty(t, videoReceived.IfRange)
		}
	}
}
