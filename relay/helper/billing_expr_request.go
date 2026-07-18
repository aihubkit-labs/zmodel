package helper

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const maxFrozenBillingRequestInputBytes = 64 << 10

var sensitiveBillingHeaders = map[string]struct{}{
	"authorization":       {},
	"cookie":              {},
	"proxy-authorization": {},
	"set-cookie":          {},
	"x-api-key":           {},
	"x-auth-token":        {},
}

func ResolveIncomingBillingExprRequestInput(c *gin.Context, info *relaycommon.RelayInfo) (billingexpr.RequestInput, error) {
	if info != nil && info.BillingRequestInput != nil {
		input := cloneRequestInput(*info.BillingRequestInput)
		merged := cloneStringMap(info.RequestHeaders)
		for k, v := range input.Headers {
			merged[k] = v
		}
		input.Headers = merged
		return input, nil
	}

	input := billingexpr.RequestInput{}
	if info != nil {
		input.Headers = cloneStringMap(info.RequestHeaders)
	}

	bodyBytes, err := readIncomingBillingExprBody(c)
	if err != nil {
		return billingexpr.RequestInput{}, err
	}
	input.Body = bodyBytes
	return input, nil
}

func BuildBillingExprRequestInputFromRequest(request dto.Request, headers map[string]string) (billingexpr.RequestInput, error) {
	input := billingexpr.RequestInput{
		Headers: cloneStringMap(headers),
	}
	if request == nil {
		return input, nil
	}

	bodyBytes, err := common.Marshal(request)
	if err != nil {
		return billingexpr.RequestInput{}, err
	}
	input.Body = bodyBytes
	return input, nil
}

// FreezeBillingExprRequestInput creates the minimal request snapshot needed to
// settle an async expression. Credentials and unrelated request content are
// never persisted into task private data.
func FreezeBillingExprRequestInput(exprStr string, input billingexpr.RequestInput) (billingexpr.RequestInput, error) {
	references, err := billingexpr.ReferencedRequestInput(exprStr)
	if err != nil {
		return billingexpr.RequestInput{}, err
	}

	frozen := billingexpr.RequestInput{}
	if len(references.Headers) > 0 {
		headers := make(map[string]string, len(input.Headers))
		for key, value := range input.Headers {
			headers[strings.ToLower(strings.TrimSpace(key))] = value
		}
		frozen.Headers = make(map[string]string, len(references.Headers))
		for _, key := range references.Headers {
			if _, sensitive := sensitiveBillingHeaders[key]; sensitive {
				return billingexpr.RequestInput{}, fmt.Errorf("header(%q) cannot be persisted for async billing settlement", key)
			}
			if value := strings.TrimSpace(headers[key]); value != "" {
				frozen.Headers[key] = value
			}
		}
	}

	if len(references.Params) > 0 {
		frozen.Params = make(map[string]any, len(references.Params))
		for _, path := range references.Params {
			if value, ok := input.Params[path]; ok {
				frozen.Params[path] = value
				continue
			}
			result := gjson.GetBytes(input.Body, path)
			if result.Exists() {
				frozen.Params[path] = result.Value()
			}
		}
	}

	encoded, err := common.Marshal(frozen)
	if err != nil {
		return billingexpr.RequestInput{}, err
	}
	if len(encoded) > maxFrozenBillingRequestInputBytes {
		return billingexpr.RequestInput{}, fmt.Errorf("async billing request snapshot exceeds %d bytes", maxFrozenBillingRequestInputBytes)
	}
	return frozen, nil
}

func readIncomingBillingExprBody(c *gin.Context) ([]byte, error) {
	if c == nil || c.Request == nil || !isJSONContentType(c.Request.Header.Get("Content-Type")) {
		return nil, nil
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	return storage.Bytes()
}

func cloneRequestInput(src billingexpr.RequestInput) billingexpr.RequestInput {
	input := billingexpr.RequestInput{
		Headers: cloneStringMap(src.Headers),
	}
	if len(src.Body) > 0 {
		input.Body = append([]byte(nil), src.Body...)
	}
	if len(src.Params) > 0 {
		input.Params = make(map[string]any, len(src.Params))
		for key, value := range src.Params {
			input.Params[key] = value
		}
	}
	return input
}

func isJSONContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(contentType, "application/json")
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		if strings.TrimSpace(key) == "" {
			continue
		}
		dst[key] = value
	}
	return dst
}
