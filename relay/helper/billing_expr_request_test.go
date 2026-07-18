package helper

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestResolveIncomingBillingExprRequestInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")

	body := []byte(`{"service_tier":"fast"}`)
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
	ctx.Set(common.KeyRequestBody, body)

	info := &relaycommon.RelayInfo{
		RequestHeaders: map[string]string{"Content-Type": "application/json"},
	}

	input, err := ResolveIncomingBillingExprRequestInput(ctx, info)
	require.NoError(t, err)
	require.Equal(t, body, input.Body)
	require.Equal(t, "application/json", input.Headers["Content-Type"])
}

func TestBuildBillingExprRequestInputFromRequest(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:  "gemini-3.1-pro-preview",
		Stream: lo.ToPtr(true),
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hi",
			},
		},
		MaxTokens: lo.ToPtr(uint(3000)),
	}

	input, err := BuildBillingExprRequestInputFromRequest(request, map[string]string{
		"Content-Type": "application/json",
		"X-Test":       "1",
	})
	require.NoError(t, err)
	require.Equal(t, "application/json", input.Headers["Content-Type"])
	require.Equal(t, "1", input.Headers["X-Test"])
	require.True(t, gjson.GetBytes(input.Body, "stream").Bool())
	require.Equal(t, "user", gjson.GetBytes(input.Body, "messages.0.role").String())
	require.Equal(t, float64(3000), gjson.GetBytes(input.Body, "max_tokens").Float())
}

func TestFreezeBillingExprRequestInputKeepsOnlyReferencedValues(t *testing.T) {
	input := billingexpr.RequestInput{
		Headers: map[string]string{
			"Anthropic-Beta": "fast-mode-2026-02-01",
			"X-Unused":       "must-not-be-persisted",
			"Authorization":  "Bearer secret",
		},
		Body: []byte(`{"service_tier":"fast","prompt":"large private prompt","nested":{"enabled":true}}`),
	}

	frozen, err := FreezeBillingExprRequestInput(
		`param("service_tier") == "fast" && header("anthropic-beta") != "" ? tier("fast", p * 2) : tier("base", p)`,
		input,
	)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"anthropic-beta": "fast-mode-2026-02-01"}, frozen.Headers)
	require.Equal(t, map[string]any{"service_tier": "fast"}, frozen.Params)
	require.Empty(t, frozen.Body)

	cost, trace, err := billingexpr.RunExprWithRequest(
		`param("service_tier") == "fast" && header("anthropic-beta") != "" ? tier("fast", p * 2) : tier("base", p)`,
		billingexpr.TokenParams{P: 10},
		frozen,
	)
	require.NoError(t, err)
	require.Equal(t, float64(20), cost)
	require.Equal(t, "fast", trace.MatchedTier)

	refrozen, err := FreezeBillingExprRequestInput(
		`param("service_tier") == "fast" && header("anthropic-beta") != "" ? tier("fast", p * 2) : tier("base", p)`,
		frozen,
	)
	require.NoError(t, err)
	require.Equal(t, frozen, refrozen)
}

func TestFreezeBillingExprRequestInputRejectsSensitiveHeader(t *testing.T) {
	_, err := FreezeBillingExprRequestInput(
		`header("Authorization") != "" ? tier("auth", p * 2) : tier("base", p)`,
		billingexpr.RequestInput{Headers: map[string]string{"Authorization": "Bearer secret"}},
	)
	require.ErrorContains(t, err, "cannot be persisted")
}

func TestFreezeBillingExprRequestInputRejectsDynamicReferences(t *testing.T) {
	_, err := FreezeBillingExprRequestInput(
		`header(param("header_name")) != "" ? tier("dynamic", p * 2) : tier("base", p)`,
		billingexpr.RequestInput{Body: []byte(`{"header_name":"x-mode"}`)},
	)
	require.ErrorContains(t, err, "literal string argument")
}
