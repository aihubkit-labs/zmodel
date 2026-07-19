package billingexpr

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strings"

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
	P    float64 // prompt tokens (text) — auto-excludes sub-categories priced separately
	C    float64 // completion tokens (text) — auto-excludes sub-categories priced separately
	Len  float64 // total input context length for tier conditions (non-Claude: raw prompt_tokens; Claude: text + cache read + cache creation)
	CR   float64 // cache read (hit) tokens
	CC   float64 // cache creation tokens (5-min TTL for Claude, generic for others)
	CC1h float64 // cache creation tokens — 1-hour TTL (Claude only)
	Img  float64 // image input tokens
	ImgO float64 // image output tokens
	AI   float64 // audio input tokens
	AO   float64 // audio output tokens
}

// BillingDimensions contains validated, normalized non-token dimensions used
// by media billing expressions. Request and provider-specific parsing happens
// before values reach this type.
type BillingDimensions struct {
	Units          float64 `json:"units"`
	Seconds        float64 `json:"seconds"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	Quality        string  `json:"quality"`
	ResolutionTier string  `json:"resolution_tier"`
	ImageSizeTier  string  `json:"image_size_tier"`
	ImageSize      string  `json:"image_size"`
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

// MergeBillingDimensions overlays non-zero/non-empty actual values onto a
// frozen estimate. Providers frequently return only a subset of dimensions at
// completion time, so missing actual fields retain the validated request value.
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
