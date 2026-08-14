package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failOnStreamPayloadWriter struct {
	gin.ResponseWriter
	needle string
}

func (w *failOnStreamPayloadWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), w.needle) {
		return 0, io.ErrClosedPipe
	}
	return w.ResponseWriter.Write(p)
}

func TestOaiStreamHandlerFinalWriteFailureOverridesDone(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Writer = &failOnStreamPayloadWriter{ResponseWriter: c.Writer, needle: `"b64_json":"image"`}

	body := strings.Join([]string{
		`data: {"id":"chatcmpl-image","object":"chat.completion.chunk","created":1,"model":"image-model","choices":[{"index":0,"delta":{"content":"![image](data:image/png;base64,AAAA)","b64_json":"image"}}]}`,
		`data: [DONE]`,
		``,
	}, "\n")
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "image-model",
		},
		RelayFormat: types.RelayFormatOpenAI,
		RelayMode:   relayconstant.RelayModeChatCompletions,
		DisablePing: true,
	}

	usage, relayErr := OaiStreamHandler(c, info, resp)

	require.Nil(t, relayErr)
	require.NotNil(t, usage)
	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonHandlerStop, info.StreamStatus.EndReason)
	assert.ErrorIs(t, info.StreamStatus.EndError, io.ErrClosedPipe)
	assert.NotContains(t, recorder.Body.String(), "data: [DONE]")
}
