package billingexpr

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

type RequestInput struct {
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"body,omitempty"`
	Params  map[string]any    `json:"params,omitempty"`
}

// TokenParams holds all token dimensions passed into an Expr evaluation.
// Fields beyond P and C are optional — when absent they default to 0,
// which means cache-unaware expressions keep working unchanged.
type TokenParams struct {
	P     float64 // prompt tokens (text) — auto-excludes sub-categories priced separately
	C     float64 // completion tokens (text) — auto-excludes sub-categories priced separately
	Len   float64 // total input context length for tier conditions (non-Claude: raw prompt_tokens; Claude: text + cache read + cache creation)
	CR    float64 // cache read (hit) tokens
	CC    float64 // cache creation tokens (5-min TTL for Claude, generic for others)
	CC1h  float64 // cache creation tokens — 1-hour TTL (Claude only)
	Img   float64 // image input tokens
	ImgO  float64 // image output tokens
	AI    float64 // audio input tokens
	AO    float64 // audio output tokens
	Total float64 // provider-reported total tokens for asynchronous media tasks
	// EvaluationTime pins time-based tier conditions to the request's pricing
	// snapshot. It is intentionally excluded from JSON persistence.
	EvaluationTime *time.Time `json:"-"`
}

// TokenUsage is the provider-neutral usage contract for asynchronous tasks.
// Pointer fields distinguish an omitted upstream value from an explicit zero.
type TokenUsage struct {
	InputTokens  *int64 `json:"input_tokens,omitempty"`
	OutputTokens *int64 `json:"output_tokens,omitempty"`
	TotalTokens  *int64 `json:"total_tokens,omitempty"`
}

func (u *TokenUsage) Params() (TokenParams, error) {
	if u == nil {
		return TokenParams{}, nil
	}
	params := TokenParams{}
	values := []struct {
		name   string
		value  *int64
		target *float64
	}{
		{name: "input_tokens", value: u.InputTokens, target: &params.P},
		{name: "output_tokens", value: u.OutputTokens, target: &params.C},
		{name: "total_tokens", value: u.TotalTokens, target: &params.Total},
	}
	for _, item := range values {
		if item.value == nil {
			continue
		}
		if *item.value < 0 {
			return TokenParams{}, fmt.Errorf("task billing usage %s must be non-negative", item.name)
		}
		*item.target = float64(*item.value)
	}
	return params, nil
}

func (u *TokenUsage) Provides(variable string) bool {
	if u == nil {
		return false
	}
	switch variable {
	case "p":
		return u.InputTokens != nil
	case "c":
		return u.OutputTokens != nil
	case "total":
		return u.TotalTokens != nil
	default:
		return true
	}
}

type EvaluationPhase int

const (
	EvaluationPhaseActual EvaluationPhase = iota
	EvaluationPhaseEstimate
)

// BillingDimensions contains validated, normalized non-token dimensions used
// by media billing expressions. Request and provider-specific parsing happens
// before values reach this type.
type BillingDimensions struct {
	Units               float64 `json:"units"`
	Seconds             float64 `json:"seconds"`
	Width               float64 `json:"width"`
	Height              float64 `json:"height"`
	ReferenceImageCount float64 `json:"reference_image_count"`
	ReferenceVideoCount float64 `json:"reference_video_count"`
	ReferenceAudioCount float64 `json:"reference_audio_count"`
	Quality             string  `json:"quality"`
	ResolutionTier      string  `json:"resolution_tier"`
	ImageSizeTier       string  `json:"image_size_tier"`
	ImageSize           string  `json:"image_size"`
}

// ValidateBillingDimensions checks the trusted dimensions required by an
// expression. A referenced dimension must not silently use its zero value,
// because that can select an unintended fallback pricing tier.
func ValidateBillingDimensions(dimensions BillingDimensions, usedVars map[string]bool) error {
	numeric := map[string]float64{
		"units":   dimensions.Units,
		"seconds": dimensions.Seconds,
		"width":   dimensions.Width,
		"height":  dimensions.Height,
	}
	for name, value := range numeric {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("billing dimension %s must be a finite non-negative number", name)
		}
		if usedVars[name] && value == 0 {
			return fmt.Errorf("billing dimension %s is required by the expression", name)
		}
	}

	counts := map[string]float64{
		"reference_image_count": dimensions.ReferenceImageCount,
		"reference_video_count": dimensions.ReferenceVideoCount,
		"reference_audio_count": dimensions.ReferenceAudioCount,
	}
	for name, value := range counts {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || math.Trunc(value) != value {
			return fmt.Errorf("billing dimension %s must be a non-negative integer", name)
		}
	}

	text := map[string]string{
		"quality":         dimensions.Quality,
		"resolution_tier": dimensions.ResolutionTier,
		"image_size_tier": dimensions.ImageSizeTier,
		"image_size":      dimensions.ImageSize,
	}
	for name, value := range text {
		if usedVars[name] && strings.TrimSpace(value) == "" {
			return fmt.Errorf("billing dimension %s is required by the expression", name)
		}
	}
	return nil
}

// MergeBillingDimensions overlays non-zero/non-empty actual output values onto
// a frozen estimate. Reference media counts are immutable request dimensions,
// so they always retain the values captured at submission time.
func MergeBillingDimensions(estimated, actual BillingDimensions) BillingDimensions {
	merged := estimated
	if actual.Units > 0 {
		merged.Units = actual.Units
	}
	if actual.Seconds > 0 {
		merged.Seconds = actual.Seconds
	}
	if actual.Width > 0 {
		merged.Width = actual.Width
	}
	if actual.Height > 0 {
		merged.Height = actual.Height
	}
	if actual.Quality != "" {
		merged.Quality = actual.Quality
	}
	if actual.ResolutionTier != "" {
		merged.ResolutionTier = actual.ResolutionTier
	}
	if actual.ImageSizeTier != "" {
		merged.ImageSizeTier = actual.ImageSizeTier
	}
	if actual.ImageSize != "" {
		merged.ImageSize = actual.ImageSize
	}
	return merged
}

// TraceResult holds side-channel info captured by the tier() function
// during Expr execution. This replaces the old Breakdown mechanism —
// the Expr itself is the single source of truth for billing logic.
type TraceResult struct {
	MatchedTier string  `json:"matched_tier"`
	Cost        float64 `json:"cost"`
}

// BillingSnapshot captures the billing rule state frozen at pre-consume time.
// It is fully serializable and contains no compiled program pointers.
type BillingSnapshot struct {
	BillingMode               string            `json:"billing_mode"`
	ModelName                 string            `json:"model_name"`
	ExprString                string            `json:"expr_string"`
	ExprHash                  string            `json:"expr_hash"`
	GroupRatio                float64           `json:"group_ratio"`
	EstimatedPromptTokens     int               `json:"estimated_prompt_tokens"`
	EstimatedCompletionTokens int               `json:"estimated_completion_tokens"`
	EstimatedQuotaBeforeGroup float64           `json:"estimated_quota_before_group"`
	EstimatedQuotaAfterGroup  int               `json:"estimated_quota_after_group"`
	EstimatedTier             string            `json:"estimated_tier"`
	QuotaPerUnit              float64           `json:"quota_per_unit"`
	ExprVersion               int               `json:"expr_version"`
	EvaluationTime            *time.Time        `json:"evaluation_time,omitempty"`
	EstimatedDimensions       BillingDimensions `json:"estimated_dimensions,omitempty"`
}

// TieredResult holds everything needed after running tiered settlement.
type TieredResult struct {
	ActualQuotaBeforeGroup float64           `json:"actual_quota_before_group"`
	ActualQuotaAfterGroup  int               `json:"actual_quota_after_group"`
	MatchedTier            string            `json:"matched_tier"`
	CrossedTier            bool              `json:"crossed_tier"`
	ActualDimensions       BillingDimensions `json:"actual_dimensions,omitempty"`
	// Clamp records an int32 saturation event during quota conversion so the
	// caller can surface it on the consume log for admin auditing. Nil when no
	// clamping occurred. Not serialized: the marker is attached separately via
	// the shared quota-saturation audit path.
	Clamp *common.QuotaClamp `json:"-"`
}

// ExprHashString returns the SHA-256 hex digest of an expression string.
func ExprHashString(expr string) string {
	h := sha256.Sum256([]byte(expr))
	return fmt.Sprintf("%x", h)
}
