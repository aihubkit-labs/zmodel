package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	TaskUpstreamHTTPRequestContextKey = "task_upstream_http_request"
	TaskUpstreamHTTPTraceContextKey   = "task_upstream_http_trace"
	UpstreamHTTPTraceBodyLimit        = 64 * 1024
	upstreamHTTPRedactedValue         = "[REDACTED]"
)

type UpstreamRequestError struct {
	Request *http.Request
	Err     error
}

func (e *UpstreamRequestError) Error() string {
	if e == nil || e.Err == nil {
		return "upstream request failed"
	}
	return e.Err.Error()
}

func (e *UpstreamRequestError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func DoTaskRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, &UpstreamRequestError{Request: req, Err: err}
	}
	return resp, nil
}

func WrapUpstreamRequestError(req *http.Request, err error) error {
	if err == nil {
		return nil
	}
	return &UpstreamRequestError{Request: req, Err: err}
}

func UpstreamHTTPTransportError(err error) *dto.TaskHTTPMessage {
	if err == nil {
		return nil
	}
	return &dto.TaskHTTPMessage{Error: sanitizeUpstreamHTTPText(err.Error())}
}

var (
	upstreamHTTPSensitiveJSONValuePattern = regexp.MustCompile(`(?i)("(?:key|[^"\\]*(?:authorization|cookie|token|secret|signature|credential|password|passwd|api[_-]?key|subscription[_-]?key)[^"\\]*)"\s*:\s*)"(?:\\.|[^"\\])*"`)
	upstreamHTTPURLPattern                = regexp.MustCompile(`https?://[^\s"'<>]+`)
)

func CaptureUpstreamHTTPRequest(req *http.Request) *dto.TaskHTTPMessage {
	if req == nil {
		return nil
	}

	message := UpstreamHTTPRequestMetadata(req)
	if req.Body == nil {
		return message
	}

	originalBody := req.Body
	prefix, err := io.ReadAll(io.LimitReader(originalBody, UpstreamHTTPTraceBodyLimit+1))
	req.Body = &replayUpstreamHTTPBody{
		prefix:    bytes.NewReader(prefix),
		original:  originalBody,
		replayErr: err,
	}
	if err != nil {
		message.Body, message.BodyTruncated = sanitizeUpstreamHTTPBody(prefix)
		message.Error = UpstreamHTTPTransportError(err).Error
		return message
	}
	message.Body, message.BodyTruncated = sanitizeUpstreamHTTPBody(prefix)
	return message
}

func CaptureUpstreamHTTPResponse(resp *http.Response) *dto.TaskHTTPMessage {
	if resp == nil {
		return nil
	}

	message := UpstreamHTTPResponseFromBody(resp, nil)
	if resp.Body == nil {
		return message
	}

	originalBody := resp.Body
	prefix, err := io.ReadAll(io.LimitReader(originalBody, UpstreamHTTPTraceBodyLimit+1))
	resp.Body = &replayUpstreamHTTPBody{
		prefix:    bytes.NewReader(prefix),
		original:  originalBody,
		replayErr: err,
	}
	if err != nil {
		message.Body, message.BodyTruncated = sanitizeUpstreamHTTPBody(prefix)
		message.Error = UpstreamHTTPTransportError(err).Error
		return message
	}
	message.Body, message.BodyTruncated = sanitizeUpstreamHTTPBody(prefix)
	return message
}

func UpstreamHTTPRequestMetadata(req *http.Request) *dto.TaskHTTPMessage {
	if req == nil {
		return nil
	}
	message := &dto.TaskHTTPMessage{
		Method:   req.Method,
		URL:      SanitizeURLForLog(req.URL.String()),
		Protocol: req.Proto,
		Headers:  sanitizeUpstreamHTTPRequestHeaders(req),
	}
	if message.Protocol == "" {
		message.Protocol = "HTTP/1.1"
	}
	return message
}

func UpstreamHTTPResponseFromBody(resp *http.Response, body []byte) *dto.TaskHTTPMessage {
	if resp == nil {
		return nil
	}
	headers := sanitizeUpstreamHTTPHeaders(resp.Header)
	if headers == nil {
		headers = make(map[string]string)
	}
	if resp.ContentLength > 0 && resp.Header.Get("Content-Length") == "" {
		headers["Content-Length"] = strconv.FormatInt(resp.ContentLength, 10)
	}
	if len(resp.TransferEncoding) > 0 && resp.Header.Get("Transfer-Encoding") == "" {
		headers["Transfer-Encoding"] = strings.Join(resp.TransferEncoding, ", ")
	}
	message := &dto.TaskHTTPMessage{
		Protocol:   resp.Proto,
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    headers,
	}
	if message.Protocol == "" {
		message.Protocol = "HTTP/1.1"
	}
	if message.Status == "" {
		message.Status = strings.TrimSpace(fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode)))
	}
	if body != nil {
		message.Body, message.BodyTruncated = sanitizeUpstreamHTTPBody(body)
	}
	return message
}

type replayUpstreamHTTPBody struct {
	prefix      *bytes.Reader
	original    io.ReadCloser
	replayErr   error
	errReturned bool
}

func (b *replayUpstreamHTTPBody) Read(p []byte) (int, error) {
	if b.prefix.Len() > 0 {
		return b.prefix.Read(p)
	}
	if b.replayErr != nil && !b.errReturned {
		b.errReturned = true
		return 0, b.replayErr
	}
	return b.original.Read(p)
}

func (b *replayUpstreamHTTPBody) Close() error {
	return b.original.Close()
}

func sanitizeUpstreamHTTPHeaders(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	sanitized := make(map[string]string, len(headers))
	for name, values := range headers {
		value := strings.Join(values, ", ")
		if isSensitiveUpstreamHTTPField(name) {
			value = upstreamHTTPRedactedValue
		} else {
			value = sanitizeUpstreamHTTPText(value)
		}
		sanitized[name] = value
	}
	return sanitized
}

func sanitizeUpstreamHTTPRequestHeaders(req *http.Request) map[string]string {
	headers := sanitizeUpstreamHTTPHeaders(req.Header)
	if headers == nil {
		headers = make(map[string]string)
	}

	host := req.Host
	if host == "" && req.URL != nil {
		host = req.URL.Host
	}
	if host != "" && req.Header.Get("Host") == "" {
		headers["Host"] = sanitizeUpstreamHTTPText(host)
	}
	if req.ContentLength > 0 && req.Header.Get("Content-Length") == "" {
		headers["Content-Length"] = strconv.FormatInt(req.ContentLength, 10)
	}
	if len(req.TransferEncoding) > 0 && req.Header.Get("Transfer-Encoding") == "" {
		headers["Transfer-Encoding"] = strings.Join(req.TransferEncoding, ", ")
	}
	return headers
}

func sanitizeUpstreamHTTPBody(body []byte) (string, bool) {
	truncated := len(body) > UpstreamHTTPTraceBodyLimit
	if truncated {
		body = body[:UpstreamHTTPTraceBodyLimit]
	}

	var value any
	if !truncated && rootcommon.Unmarshal(body, &value) == nil {
		value = sanitizeUpstreamHTTPJSONValue(value)
		if sanitized, err := rootcommon.Marshal(value); err == nil {
			return string(sanitized), false
		}
	}
	return sanitizeUpstreamHTTPText(string(body)), truncated
}

func sanitizeUpstreamHTTPJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isSensitiveUpstreamHTTPField(key) {
				typed[key] = upstreamHTTPRedactedValue
				continue
			}
			typed[key] = sanitizeUpstreamHTTPJSONValue(child)
		}
		return typed
	case []any:
		for index, child := range typed {
			typed[index] = sanitizeUpstreamHTTPJSONValue(child)
		}
		return typed
	case string:
		return SanitizeURLForLog(typed)
	default:
		return value
	}
}

func sanitizeUpstreamHTTPText(value string) string {
	value = upstreamHTTPSensitiveJSONValuePattern.ReplaceAllString(value, `${1}"`+upstreamHTTPRedactedValue+`"`)
	return upstreamHTTPURLPattern.ReplaceAllStringFunc(value, SanitizeURLForLog)
}

func isSensitiveUpstreamHTTPField(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "authorization", "proxy-authorization", "cookie", "set-cookie", "key", "api-key", "api_key", "apikey", "x-api-key", "x-goog-api-key", "password", "passwd", "client-secret", "client_secret":
		return true
	}
	return strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "signature") ||
		strings.Contains(normalized, "credential") ||
		strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "cookie") ||
		strings.HasSuffix(normalized, "-key") ||
		strings.HasSuffix(normalized, "_key")
}
