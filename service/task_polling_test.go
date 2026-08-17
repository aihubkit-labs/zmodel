package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type taskPollingFetchAdaptor struct {
	mu           sync.Mutex
	taskIDs      []string
	fetched      chan string
	status       model.TaskStatus
	progress     string
	resultURL    string
	blockTaskID  string
	blockStarted chan struct{}
	releaseBlock chan struct{}
	blockOnce    sync.Once
	fetchErr     error
	protocols    []dto.VideoProtocol
}

func (a *taskPollingFetchAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *taskPollingFetchAdaptor) FetchTask(_ string, _ string, body map[string]any, _ string) (*http.Response, error) {
	taskID, _ := body["task_id"].(string)
	if taskID == a.blockTaskID && a.releaseBlock != nil {
		a.blockOnce.Do(func() {
			if a.blockStarted != nil {
				close(a.blockStarted)
			}
		})
		<-a.releaseBlock
	}

	a.mu.Lock()
	a.taskIDs = append(a.taskIDs, taskID)
	if protocol, ok := body["video_protocol"].(dto.VideoProtocol); ok {
		a.protocols = append(a.protocols, protocol)
	}
	a.mu.Unlock()
	if a.fetched != nil {
		select {
		case a.fetched <- taskID:
		default:
		}
	}
	if a.fetchErr != nil {
		return nil, a.fetchErr
	}

	status := a.status
	if status == "" {
		status = model.TaskStatusInProgress
	}
	progress := a.progress
	if progress == "" {
		progress = "30%"
	}
	response := dto.TaskResponse[model.Task]{
		Code: dto.TaskSuccessCode,
		Data: model.Task{
			TaskID:     taskID,
			Status:     status,
			Progress:   progress,
			FailReason: a.resultURL,
		},
	}
	responseBody, err := common.Marshal(response)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
		Request: &http.Request{
			Method: http.MethodGet,
			URL: &url.URL{
				Scheme: "https", Host: "upstream.example", Path: "/tasks/" + taskID, RawQuery: "token=poll-secret",
			},
			Header: http.Header{"Authorization": {"Bearer poll-secret"}},
		},
	}, nil
}

func (a *taskPollingFetchAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{Status: model.TaskStatusInProgress}, nil
}

func (a *taskPollingFetchAdaptor) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return 0
}

func (a *taskPollingFetchAdaptor) fetchCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.taskIDs)
}

func (a *taskPollingFetchAdaptor) fetchedTaskIDs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.taskIDs...)
}

func seedTaskPollingChannel(t *testing.T, id int, disableSleep bool) {
	t.Helper()
	ch := &model.Channel{
		Id:     id,
		Type:   constant.ChannelTypeKling,
		Name:   "polling_channel",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	if disableSleep {
		ch.SetOtherSettings(dto.ChannelOtherSettings{DisableTaskPollingSleep: true})
	}
	require.NoError(t, model.DB.Create(ch).Error)
}

func seedPollingTask(t *testing.T, channelID int, publicID string, upstreamID string) *model.Task {
	t.Helper()
	task := &model.Task{
		TaskID:    publicID,
		Platform:  constant.TaskPlatform("kling"),
		UserId:    1,
		ChannelId: channelID,
		Action:    constant.TaskActionGenerate,
		Status:    model.TaskStatusInProgress,
		Progress:  "30%",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: upstreamID,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)
	return task
}

func TestUpdateVideoTasksDefaultSleepWaitsBetweenTasks(t *testing.T) {
	truncate(t)

	const channelID = 101
	seedTaskPollingChannel(t, channelID, false)
	first := seedPollingTask(t, channelID, "task_public_1", "upstream_1")
	second := seedPollingTask(t, channelID, "task_public_2", "upstream_2")

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := UpdateVideoTasks(ctx, constant.TaskPlatform("kling"), map[int][]string{
		channelID: {
			first.GetUpstreamTaskID(),
			second.GetUpstreamTaskID(),
		},
	}, map[string]*model.Task{
		first.GetUpstreamTaskID():  first,
		second.GetUpstreamTaskID(): second,
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, adaptor.fetchCount())
}

func TestUpdateVideoTasksCanSkipPollingSleepPerChannel(t *testing.T) {
	truncate(t)

	const channelID = 102
	seedTaskPollingChannel(t, channelID, true)
	first := seedPollingTask(t, channelID, "task_public_3", "upstream_3")
	second := seedPollingTask(t, channelID, "task_public_4", "upstream_4")

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := UpdateVideoTasks(ctx, constant.TaskPlatform("kling"), map[int][]string{
		channelID: {
			first.GetUpstreamTaskID(),
			second.GetUpstreamTaskID(),
		},
	}, map[string]*model.Task{
		first.GetUpstreamTaskID():  first,
		second.GetUpstreamTaskID(): second,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, adaptor.fetchCount())
}

func TestUpdateVideoTaskForcesTerminalProgressToComplete(t *testing.T) {
	truncate(t)

	const channelID = 103
	seedTaskPollingChannel(t, channelID, true)
	task := seedPollingTask(t, channelID, "task_public_terminal", "upstream_terminal")
	adaptor := &taskPollingFetchAdaptor{
		status:   model.TaskStatusSuccess,
		progress: "30%",
	}

	err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{
		Id:     channelID,
		Type:   constant.ChannelTypeKling,
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}, task.GetUpstreamTaskID(), map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	})
	require.NoError(t, err)

	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusSuccess), stored.Status)
	assert.Equal(t, "100%", stored.Progress)
}

func TestUpdateVideoTaskStoresSanitizedHTTPTraceOnFailure(t *testing.T) {
	truncate(t)

	const channelID = 105
	seedTaskPollingChannel(t, channelID, true)
	task := seedPollingTask(t, channelID, "task_public_failure_trace", "upstream_failure_trace")
	task.PrivateData.UpstreamHTTPTrace = &dto.TaskUpstreamHTTPTrace{
		SubmitRequest: &dto.TaskHTTPMessage{Method: http.MethodPost, URL: "https://upstream.example/v1/videos"},
	}
	require.NoError(t, model.DB.Model(task).Update("private_data", task.PrivateData).Error)
	adaptor := &taskPollingFetchAdaptor{
		status:    model.TaskStatusFailure,
		resultURL: "upstream rejected the request",
	}

	err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{
		Id: channelID, Type: constant.ChannelTypeKling, Key: "sk-test", Status: common.ChannelStatusEnabled,
	}, task.GetUpstreamTaskID(), map[string]*model.Task{task.GetUpstreamTaskID(): task})
	require.NoError(t, err)

	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace)
	assert.Equal(t, http.MethodPost, stored.PrivateData.UpstreamHTTPTrace.SubmitRequest.Method)
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace.PollRequest)
	assert.NotContains(t, stored.PrivateData.UpstreamHTTPTrace.PollRequest.URL, "poll-secret")
	assert.Equal(t, "[REDACTED]", stored.PrivateData.UpstreamHTTPTrace.PollRequest.Headers["Authorization"])
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace.PollResponse)
	assert.Equal(t, http.StatusOK, stored.PrivateData.UpstreamHTTPTrace.PollResponse.StatusCode)
	assert.Contains(t, stored.PrivateData.UpstreamHTTPTrace.PollResponse.Body, "upstream rejected the request")
}

func TestUpdateVideoTaskStoresSanitizedHTTPTraceOnSuccess(t *testing.T) {
	truncate(t)

	const channelID = 106
	seedTaskPollingChannel(t, channelID, true)
	task := seedPollingTask(t, channelID, "task_public_success_trace", "upstream_success_trace")
	task.PrivateData.UpstreamHTTPTrace = &dto.TaskUpstreamHTTPTrace{
		SubmitRequest: &dto.TaskHTTPMessage{Method: http.MethodPost},
	}
	require.NoError(t, model.DB.Model(task).Update("private_data", task.PrivateData).Error)
	adaptor := &taskPollingFetchAdaptor{status: model.TaskStatusSuccess}

	err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{
		Id: channelID, Type: constant.ChannelTypeKling, Key: "sk-test", Status: common.ChannelStatusEnabled,
	}, task.GetUpstreamTaskID(), map[string]*model.Task{task.GetUpstreamTaskID(): task})
	require.NoError(t, err)

	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace)
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace.SubmitRequest)
	assert.Equal(t, http.MethodPost, stored.PrivateData.UpstreamHTTPTrace.SubmitRequest.Method)
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace.PollRequest)
	assert.NotContains(t, stored.PrivateData.UpstreamHTTPTrace.PollRequest.URL, "poll-secret")
	assert.Equal(t, "[REDACTED]", stored.PrivateData.UpstreamHTTPTrace.PollRequest.Headers["Authorization"])
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace.PollResponse)
	assert.Equal(t, http.StatusOK, stored.PrivateData.UpstreamHTTPTrace.PollResponse.StatusCode)
}

func TestUpdateVideoTaskStoresTransportErrorWithoutFailingImmediately(t *testing.T) {
	truncate(t)

	const channelID = 107
	seedTaskPollingChannel(t, channelID, true)
	task := seedPollingTask(t, channelID, "task_public_transport_error", "upstream_transport_error")
	request := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https", Host: "upstream.example", Path: "/v1/videos/upstream_transport_error", RawQuery: "token=poll-secret",
		},
		Header: http.Header{"Authorization": {"Bearer poll-secret"}},
	}
	adaptor := &taskPollingFetchAdaptor{
		fetchErr: &relaycommon.UpstreamRequestError{Request: request, Err: context.DeadlineExceeded},
	}

	err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{
		Id: channelID, Type: constant.ChannelTypeKling, Key: "sk-test", Status: common.ChannelStatusEnabled,
	}, task.GetUpstreamTaskID(), map[string]*model.Task{task.GetUpstreamTaskID(): task})
	require.Error(t, err)

	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusInProgress), stored.Status)
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace)
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace.PollRequest)
	assert.NotContains(t, stored.PrivateData.UpstreamHTTPTrace.PollRequest.URL, "poll-secret")
	assert.Equal(t, "[REDACTED]", stored.PrivateData.UpstreamHTTPTrace.PollRequest.Headers["Authorization"])
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace.PollResponse)
	assert.Contains(t, stored.PrivateData.UpstreamHTTPTrace.PollResponse.Error, context.DeadlineExceeded.Error())
	assert.NotContains(t, stored.PrivateData.UpstreamHTTPTrace.PollResponse.Error, "poll-secret")
}

func TestS3EnabledVideoTaskSettlesWhenArchiveFails(t *testing.T) {
	truncate(t)

	const channelID = 104
	seedTaskPollingChannel(t, channelID, true)
	task := seedPollingTask(t, channelID, "task_public_s3", "upstream_s3")
	task.PrivateData.VideoS3StorageEnabled = true
	require.NoError(t, model.DB.Model(task).Update("private_data", task.PrivateData).Error)
	adaptor := &taskPollingFetchAdaptor{
		status:    model.TaskStatusSuccess,
		resultURL: "https://video.example/result.mp4",
	}

	originalArchive := ArchiveVideoTaskFunc
	archiveCalls := 0
	ArchiveVideoTaskFunc = func(_ context.Context, archivedTask *model.Task, _ *model.Channel, source VideoArchiveSource) error {
		archiveCalls++
		assert.Equal(t, task.TaskID, archivedTask.TaskID)
		assert.Equal(t, "https://video.example/result.mp4", source.URL)
		return assert.AnError
	}
	t.Cleanup(func() { ArchiveVideoTaskFunc = originalArchive })
	channel := &model.Channel{
		Id: channelID, Type: constant.ChannelTypeKling, Key: "sk-test",
		Status: common.ChannelStatusEnabled,
	}

	err := updateVideoSingleTask(context.Background(), adaptor, channel, task.GetUpstreamTaskID(), map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	})
	require.NoError(t, err)
	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusSuccess), stored.Status)
	assert.Equal(t, taskcommon.ProgressComplete, stored.Progress)
	assert.Equal(t, "https://video.example/result.mp4", stored.GetResultURL())
	assert.Equal(t, 1, archiveCalls)
}

func TestOpenAIVideoPollingDoesNotPersistDerivedOrTemporaryURLs(t *testing.T) {
	truncate(t)

	const channelID = 105
	task := seedPollingTask(t, channelID, "task_public_globalaiopc", "upstream_globalaiopc")
	task.PrivateData.BillingContext = &model.TaskBillingContext{
		VideoProtocol: dto.VideoProtocolGlobalAIOpc,
	}
	require.NoError(t, model.DB.Model(task).Update("private_data", task.PrivateData).Error)
	adaptor := &taskPollingFetchAdaptor{
		status:    model.TaskStatusSuccess,
		resultURL: "https://upstream.example/temporary.mp4",
	}
	channel := &model.Channel{
		Id: channelID, Type: constant.ChannelTypeOpenAI, Key: "sk-test",
		Status: common.ChannelStatusEnabled,
	}

	err := updateVideoSingleTask(context.Background(), adaptor, channel, task.GetUpstreamTaskID(), map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	})
	require.NoError(t, err)

	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	assert.Empty(t, stored.GetResultURL())
	require.Len(t, adaptor.protocols, 1)
	assert.Equal(t, dto.VideoProtocolGlobalAIOpc, adaptor.protocols[0])
}

func TestRedactVideoResponseBodyRemovesTemporaryUpstreamURLs(t *testing.T) {
	body := []byte(`{
		"status":"completed",
		"url":"https://upstream.example/url.mp4",
		"result_url":"https://upstream.example/result.mp4",
		"video_url":"https://upstream.example/video.mp4",
		"metadata":{"origin_video_url":"https://upstream.example/origin.mp4","size_mapping":{"resolution":"1440p"}}
	}`)

	redacted := redactVideoResponseBody(body, true)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(redacted, &payload))
	assert.NotContains(t, payload, "url")
	assert.NotContains(t, payload, "result_url")
	assert.NotContains(t, payload, "video_url")
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, metadata, "origin_video_url")
	assert.Contains(t, metadata, "size_mapping")
}

func TestUpdateVideoTasksDefaultSleepDoesNotBlockOtherChannels(t *testing.T) {
	truncate(t)

	const firstChannelID = 201
	const secondChannelID = 202
	seedTaskPollingChannel(t, firstChannelID, false)
	seedTaskPollingChannel(t, secondChannelID, false)
	firstChannelFirst := seedPollingTask(t, firstChannelID, "task_public_5", "upstream_a_1")
	firstChannelSecond := seedPollingTask(t, firstChannelID, "task_public_6", "upstream_a_2")
	secondChannelFirst := seedPollingTask(t, secondChannelID, "task_public_7", "upstream_b_1")
	secondChannelSecond := seedPollingTask(t, secondChannelID, "task_public_8", "upstream_b_2")

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := UpdateVideoTasks(ctx, constant.TaskPlatform("kling"), map[int][]string{
		firstChannelID: {
			firstChannelFirst.GetUpstreamTaskID(),
			firstChannelSecond.GetUpstreamTaskID(),
		},
		secondChannelID: {
			secondChannelFirst.GetUpstreamTaskID(),
			secondChannelSecond.GetUpstreamTaskID(),
		},
	}, map[string]*model.Task{
		firstChannelFirst.GetUpstreamTaskID():   firstChannelFirst,
		firstChannelSecond.GetUpstreamTaskID():  firstChannelSecond,
		secondChannelFirst.GetUpstreamTaskID():  secondChannelFirst,
		secondChannelSecond.GetUpstreamTaskID(): secondChannelSecond,
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.ElementsMatch(t, []string{"upstream_a_1", "upstream_b_1"}, adaptor.fetchedTaskIDs())
}

func TestUpdateVideoTasksSlowChannelDoesNotBlockOtherChannels(t *testing.T) {
	truncate(t)

	const slowChannelID = 251
	const fastChannelID = 252
	seedTaskPollingChannel(t, slowChannelID, false)
	seedTaskPollingChannel(t, fastChannelID, true)
	slowTask := seedPollingTask(t, slowChannelID, "task_public_slow", "upstream_slow_1")
	fastFirst := seedPollingTask(t, fastChannelID, "task_public_fast_1", "upstream_fast_parallel_1")
	fastSecond := seedPollingTask(t, fastChannelID, "task_public_fast_2", "upstream_fast_parallel_2")

	adaptor := &taskPollingFetchAdaptor{
		fetched:      make(chan string, 4),
		blockTaskID:  slowTask.GetUpstreamTaskID(),
		blockStarted: make(chan struct{}),
		releaseBlock: make(chan struct{}),
	}
	var releaseOnce sync.Once
	releaseBlockedTask := func() {
		releaseOnce.Do(func() {
			close(adaptor.releaseBlock)
		})
	}
	t.Cleanup(releaseBlockedTask)
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	errCh := make(chan error, 1)
	gopool.Go(func() {
		errCh <- UpdateVideoTasks(context.Background(), constant.TaskPlatform("kling"), map[int][]string{
			slowChannelID: {
				slowTask.GetUpstreamTaskID(),
			},
			fastChannelID: {
				fastFirst.GetUpstreamTaskID(),
				fastSecond.GetUpstreamTaskID(),
			},
		}, map[string]*model.Task{
			slowTask.GetUpstreamTaskID():   slowTask,
			fastFirst.GetUpstreamTaskID():  fastFirst,
			fastSecond.GetUpstreamTaskID(): fastSecond,
		})
	})

	select {
	case <-adaptor.blockStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("slow channel did not start blocking")
	}

	require.Eventually(t, func() bool {
		fetchedTaskIDs := adaptor.fetchedTaskIDs()
		return len(fetchedTaskIDs) == 2 &&
			fetchedTaskIDs[0] == fastFirst.GetUpstreamTaskID() &&
			fetchedTaskIDs[1] == fastSecond.GetUpstreamTaskID()
	}, 500*time.Millisecond, 10*time.Millisecond)

	releaseBlockedTask()
	require.NoError(t, <-errCh)
	assert.ElementsMatch(t, []string{
		slowTask.GetUpstreamTaskID(),
		fastFirst.GetUpstreamTaskID(),
		fastSecond.GetUpstreamTaskID(),
	}, adaptor.fetchedTaskIDs())
}

func TestUpdateVideoTasksMixedChannelSleepSettings(t *testing.T) {
	truncate(t)

	const sleepyChannelID = 301
	const fastChannelID = 302
	seedTaskPollingChannel(t, sleepyChannelID, false)
	seedTaskPollingChannel(t, fastChannelID, true)
	sleepyFirst := seedPollingTask(t, sleepyChannelID, "task_public_9", "upstream_sleepy_1")
	sleepySecond := seedPollingTask(t, sleepyChannelID, "task_public_10", "upstream_sleepy_2")
	fastFirst := seedPollingTask(t, fastChannelID, "task_public_11", "upstream_fast_1")
	fastSecond := seedPollingTask(t, fastChannelID, "task_public_12", "upstream_fast_2")

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := UpdateVideoTasks(ctx, constant.TaskPlatform("kling"), map[int][]string{
		sleepyChannelID: {
			sleepyFirst.GetUpstreamTaskID(),
			sleepySecond.GetUpstreamTaskID(),
		},
		fastChannelID: {
			fastFirst.GetUpstreamTaskID(),
			fastSecond.GetUpstreamTaskID(),
		},
	}, map[string]*model.Task{
		sleepyFirst.GetUpstreamTaskID():  sleepyFirst,
		sleepySecond.GetUpstreamTaskID(): sleepySecond,
		fastFirst.GetUpstreamTaskID():    fastFirst,
		fastSecond.GetUpstreamTaskID():   fastSecond,
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.ElementsMatch(t, []string{"upstream_sleepy_1", "upstream_fast_1", "upstream_fast_2"}, adaptor.fetchedTaskIDs())
}
