package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestVideoProxyMapsUpstreamFailureToBadGateway(t *testing.T) {
	testVideoProxyDownload(t, videoProxyTestCase{
		storedKey:      "stored-task-key",
		channelKey:     "current-channel-key",
		expectedKey:    "stored-task-key",
		upstreamStatus: http.StatusUnauthorized,
		expectedStatus: http.StatusBadGateway,
	})
}

func testVideoProxyDownload(t *testing.T, testCase videoProxyTestCase) {
	t.Helper()
	db := setupVideoProxyTest(t)

	upstreamRequest := make(chan videoProxyUpstreamRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequest <- videoProxyUpstreamRequest{
			Path:          r.URL.Path,
			Authorization: r.Header.Get("Authorization"),
			Range:         r.Header.Get("Range"),
			IfRange:       r.Header.Get("If-Range"),
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
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	channel := &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeOpenAI,
		Key:     testCase.channelKey,
		Name:    "FriModel OpenAI video",
		BaseURL: &baseURL,
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
		assert.Contains(t, recorder.Body.String(), "Upstream service returned status 401")
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

	received := <-upstreamRequest
	assert.Equal(t, "/v1/videos/task_frimodel_upstream/content", received.Path)
	assert.Equal(t, "Bearer "+testCase.expectedKey, received.Authorization)
	if testCase.forwardRange {
		assert.Equal(t, "bytes=0-3", received.Range)
		assert.Equal(t, `"video-etag"`, received.IfRange)
	} else {
		assert.Empty(t, received.Range)
		assert.Empty(t, received.IfRange)
	}
}
