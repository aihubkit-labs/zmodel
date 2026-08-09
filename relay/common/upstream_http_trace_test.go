package common

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type upstreamHTTPRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f upstreamHTTPRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type upstreamHTTPFailingBody struct {
	read bool
}

func (b *upstreamHTTPFailingBody) Read(p []byte) (int, error) {
	if b.read {
		return 0, io.EOF
	}
	b.read = true
	return copy(p, `{"error":"partial"}`), context.DeadlineExceeded
}

func (b *upstreamHTTPFailingBody) Close() error {
	return nil
}

func TestCaptureUpstreamHTTPRequestPreservesBodyAndRedactsCredentials(t *testing.T) {
	body := `{"model":"seedance-2.5","reference_image":"https://cdn.example/input.jpg?token=media-secret&part=1","api_key":"body-secret","key":"generic-secret"}`
	req, err := http.NewRequest(http.MethodPost, "https://upstream.example/v1/videos?X-Amz-Signature=url-secret&mode=create", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer header-secret")
	req.Header.Set("Cookie", "session=cookie-secret")
	req.Header.Set("Content-Type", "application/json")

	message := CaptureUpstreamHTTPRequest(req)
	replayedBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	assert.Equal(t, body, string(replayedBody))
	assert.Equal(t, http.MethodPost, message.Method)
	assert.Equal(t, "HTTP/1.1", message.Protocol)
	assert.Contains(t, message.URL, "X-Amz-Signature=%2A%2A%2Amasked%2A%2A%2A")
	assert.Contains(t, message.URL, "mode=create")
	assert.Equal(t, "upstream.example", message.Headers["Host"])
	assert.Equal(t, strconv.Itoa(len(body)), message.Headers["Content-Length"])
	assert.Equal(t, upstreamHTTPRedactedValue, message.Headers["Authorization"])
	assert.Equal(t, upstreamHTTPRedactedValue, message.Headers["Cookie"])
	assert.Equal(t, "application/json", message.Headers["Content-Type"])
	assert.NotContains(t, message.Body, "body-secret")
	assert.NotContains(t, message.Body, "generic-secret")
	assert.NotContains(t, message.Body, "media-secret")
	assert.Contains(t, message.Body, upstreamHTTPRedactedValue)
	assert.False(t, message.BodyTruncated)
}

func TestCaptureUpstreamHTTPResponseTruncatesBodyAndPreservesStream(t *testing.T) {
	body := strings.Repeat("x", UpstreamHTTPTraceBodyLimit+128)
	resp := &http.Response{
		StatusCode:    http.StatusForbidden,
		Status:        "403 Forbidden",
		Proto:         "HTTP/2.0",
		ContentLength: int64(len(body)),
		Header: http.Header{
			"Set-Cookie": {"session=response-secret"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	message := CaptureUpstreamHTTPResponse(resp)
	replayedBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, body, string(replayedBody))
	assert.Equal(t, http.StatusForbidden, message.StatusCode)
	assert.Equal(t, "403 Forbidden", message.Status)
	assert.Equal(t, "HTTP/2.0", message.Protocol)
	assert.Equal(t, strconv.Itoa(len(body)), message.Headers["Content-Length"])
	assert.Equal(t, upstreamHTTPRedactedValue, message.Headers["Set-Cookie"])
	assert.Len(t, message.Body, UpstreamHTTPTraceBodyLimit)
	assert.True(t, message.BodyTruncated)
}

func TestDoTaskRequestPreservesRequestWhenTransportFails(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://upstream.example/v1/videos/task-1?token=request-secret", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer request-secret")
	client := &http.Client{Transport: upstreamHTTPRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("context deadline exceeded")
	})}

	resp, err := DoTaskRequest(client, req)

	assert.Nil(t, resp)
	var requestErr *UpstreamRequestError
	require.ErrorAs(t, err, &requestErr)
	requestMessage := UpstreamHTTPRequestMetadata(requestErr.Request)
	assert.NotContains(t, requestMessage.URL, "request-secret")
	assert.Equal(t, upstreamHTTPRedactedValue, requestMessage.Headers["Authorization"])
	errorMessage := UpstreamHTTPTransportError(requestErr)
	assert.Contains(t, errorMessage.Error, "context deadline exceeded")
	assert.NotContains(t, errorMessage.Error, "request-secret")
}

func TestCaptureUpstreamHTTPResponsePreservesBodyReadError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       &upstreamHTTPFailingBody{},
	}

	message := CaptureUpstreamHTTPResponse(resp)
	replayedBody, err := io.ReadAll(resp.Body)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.JSONEq(t, `{"error":"partial"}`, string(replayedBody))
	assert.Equal(t, http.StatusBadGateway, message.StatusCode)
	assert.JSONEq(t, `{"error":"partial"}`, message.Body)
	assert.Equal(t, context.DeadlineExceeded.Error(), message.Error)
}

func TestUpstreamHTTPResponseFromBodyBuildsMissingStatusLineFields(t *testing.T) {
	message := UpstreamHTTPResponseFromBody(&http.Response{
		StatusCode: http.StatusBadGateway,
	}, nil)

	assert.Equal(t, "HTTP/1.1", message.Protocol)
	assert.Equal(t, "502 Bad Gateway", message.Status)
}
