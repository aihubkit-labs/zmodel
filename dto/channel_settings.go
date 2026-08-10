package dto

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/constant"
)

type ChannelSettings struct {
	ForceFormat              bool                            `json:"force_format,omitempty"`
	ThinkingToContent        bool                            `json:"thinking_to_content,omitempty"`
	Proxy                    string                          `json:"proxy"`
	VideoContentProxyEnabled bool                            `json:"video_content_proxy_enabled,omitempty"`
	VideoS3StorageEnabled    bool                            `json:"video_s3_storage_enabled,omitempty"`
	VideoS3Preferred         bool                            `json:"video_s3_preferred,omitempty"`
	VideoProtocol            VideoProtocol                   `json:"video_protocol,omitempty"`
	VideoModelCapabilities   map[string]VideoModelCapability `json:"video_model_capabilities,omitempty"`
	PassThroughBodyEnabled   bool                            `json:"pass_through_body_enabled,omitempty"`
	SystemPrompt             string                          `json:"system_prompt,omitempty"`
	SystemPromptOverride     bool                            `json:"system_prompt_override,omitempty"`
}

type VideoModelCapability struct {
	Resolutions                           []string `json:"resolutions,omitempty"`
	Ratios                                []string `json:"ratios,omitempty"`
	MaxReferenceImages                    *int     `json:"max_reference_images,omitempty"`
	MaxReferenceVideos                    *int     `json:"max_reference_videos,omitempty"`
	MaxReferenceAudios                    *int     `json:"max_reference_audios,omitempty"`
	MinDurationSeconds                    *int     `json:"min_duration_seconds,omitempty"`
	MaxDurationSeconds                    *int     `json:"max_duration_seconds,omitempty"`
	SupportsGenerateAudio                 *bool    `json:"supports_generate_audio,omitempty"`
	GenerateAudioRequired                 *bool    `json:"generate_audio_required,omitempty"`
	SupportsFirstFrame                    *bool    `json:"supports_first_frame,omitempty"`
	SupportsLastFrame                     *bool    `json:"supports_last_frame,omitempty"`
	LastFrameRequiresFirstFrame           *bool    `json:"last_frame_requires_first_frame,omitempty"`
	ReferenceImagesIncompatibleWithFrames *bool    `json:"reference_images_incompatible_with_frames,omitempty"`
	AudioReferenceRequiresVisualReference *bool    `json:"audio_reference_requires_visual_reference,omitempty"`
}

const MaxVideoReferenceCount = 64
const MaxVideoModelResolutions = 32
const MaxVideoModelRatios = 32
const MaxVideoDurationSeconds = 3600

type VideoProtocol string

const (
	VideoProtocolOpenAI            VideoProtocol = "openai_video"
	VideoProtocolSeedanceMegabyAI  VideoProtocol = "seedance(megabyai)"
	VideoProtocolMinimaxH3MegabyAI VideoProtocol = "minimax-h3(megabyai)"
	VideoProtocolAgnesVideoV2      VideoProtocol = "agnes_video_v2"
)

func (s ChannelSettings) ValidateVideoRequestSettings() error {
	switch s.VideoProtocol {
	case "", VideoProtocolOpenAI, VideoProtocolSeedanceMegabyAI, VideoProtocolMinimaxH3MegabyAI, VideoProtocolAgnesVideoV2:
	default:
		return fmt.Errorf("unsupported video_protocol %q", s.VideoProtocol)
	}
	if s.VideoProtocol != "" && len(s.VideoModelCapabilities) == 0 {
		return fmt.Errorf("video_model_capabilities is required when video_protocol is configured")
	}

	normalizedModels := make(map[string]struct{}, len(s.VideoModelCapabilities))
	for modelName, capability := range s.VideoModelCapabilities {
		normalizedModel := strings.ToLower(strings.TrimSpace(modelName))
		if normalizedModel == "" {
			return fmt.Errorf("video_model_capabilities contains an empty model ID")
		}
		if len(normalizedModel) > 128 {
			return fmt.Errorf("video model ID %q exceeds 128 characters", modelName)
		}
		if _, exists := normalizedModels[normalizedModel]; exists {
			return fmt.Errorf("video_model_capabilities contains duplicate model ID %q", modelName)
		}
		normalizedModels[normalizedModel] = struct{}{}

		if len(capability.Resolutions) == 0 {
			return fmt.Errorf("video model %q must configure at least one resolution", modelName)
		}
		if len(capability.Resolutions) > MaxVideoModelResolutions {
			return fmt.Errorf("video model %q cannot configure more than %d resolutions", modelName, MaxVideoModelResolutions)
		}
		seenResolutions := make(map[string]struct{}, len(capability.Resolutions))
		for _, resolution := range capability.Resolutions {
			normalizedResolution := strings.ToLower(strings.TrimSpace(resolution))
			if normalizedResolution == "" {
				return fmt.Errorf("video model %q contains an empty resolution", modelName)
			}
			if len(normalizedResolution) > 64 {
				return fmt.Errorf("video model %q resolution %q exceeds 64 characters", modelName, resolution)
			}
			if _, exists := seenResolutions[normalizedResolution]; exists {
				return fmt.Errorf("video model %q contains duplicate resolution %q", modelName, resolution)
			}
			seenResolutions[normalizedResolution] = struct{}{}
		}
		if len(capability.Ratios) > MaxVideoModelRatios {
			return fmt.Errorf("video model %q cannot configure more than %d ratios", modelName, MaxVideoModelRatios)
		}
		seenRatios := make(map[string]struct{}, len(capability.Ratios))
		for _, ratio := range capability.Ratios {
			normalizedRatio := strings.ToLower(strings.TrimSpace(ratio))
			if normalizedRatio == "" {
				return fmt.Errorf("video model %q contains an empty ratio", modelName)
			}
			if len(normalizedRatio) > 64 {
				return fmt.Errorf("video model %q ratio %q exceeds 64 characters", modelName, ratio)
			}
			if _, exists := seenRatios[normalizedRatio]; exists {
				return fmt.Errorf("video model %q contains duplicate ratio %q", modelName, ratio)
			}
			seenRatios[normalizedRatio] = struct{}{}
		}

		for _, limit := range []struct {
			name  string
			value *int
		}{
			{name: "max_reference_images", value: capability.MaxReferenceImages},
			{name: "max_reference_videos", value: capability.MaxReferenceVideos},
			{name: "max_reference_audios", value: capability.MaxReferenceAudios},
		} {
			if limit.value == nil {
				return fmt.Errorf("video model %q must configure %s", modelName, limit.name)
			}
			if *limit.value < 0 || *limit.value > MaxVideoReferenceCount {
				return fmt.Errorf("video model %q %s must be between 0 and %d", modelName, limit.name, MaxVideoReferenceCount)
			}
		}
		if s.VideoProtocol == VideoProtocolAgnesVideoV2 &&
			capability.MaxReferenceImages != nil && *capability.MaxReferenceImages > 1 {
			return fmt.Errorf("video model %q max_reference_images cannot exceed 1 for Agnes Video V2", modelName)
		}
		if s.VideoProtocol == VideoProtocolSeedanceMegabyAI || s.VideoProtocol == VideoProtocolMinimaxH3MegabyAI {
			if capability.MinDurationSeconds == nil {
				return fmt.Errorf("video model %q must configure min_duration_seconds", modelName)
			}
			if capability.MaxDurationSeconds == nil {
				return fmt.Errorf("video model %q must configure max_duration_seconds", modelName)
			}
			if *capability.MinDurationSeconds < 1 || *capability.MinDurationSeconds > MaxVideoDurationSeconds {
				return fmt.Errorf("video model %q min_duration_seconds must be between 1 and %d", modelName, MaxVideoDurationSeconds)
			}
			if *capability.MaxDurationSeconds < 1 || *capability.MaxDurationSeconds > MaxVideoDurationSeconds {
				return fmt.Errorf("video model %q max_duration_seconds must be between 1 and %d", modelName, MaxVideoDurationSeconds)
			}
			if *capability.MinDurationSeconds > *capability.MaxDurationSeconds {
				return fmt.Errorf("video model %q min_duration_seconds cannot exceed max_duration_seconds", modelName)
			}
		}
		if s.VideoProtocol == VideoProtocolMinimaxH3MegabyAI {
			if len(capability.Ratios) == 0 {
				return fmt.Errorf("video model %q must configure at least one ratio", modelName)
			}
			for _, field := range []struct {
				name  string
				value *bool
			}{
				{name: "supports_generate_audio", value: capability.SupportsGenerateAudio},
				{name: "generate_audio_required", value: capability.GenerateAudioRequired},
				{name: "supports_first_frame", value: capability.SupportsFirstFrame},
				{name: "supports_last_frame", value: capability.SupportsLastFrame},
				{name: "last_frame_requires_first_frame", value: capability.LastFrameRequiresFirstFrame},
				{name: "reference_images_incompatible_with_frames", value: capability.ReferenceImagesIncompatibleWithFrames},
				{name: "audio_reference_requires_visual_reference", value: capability.AudioReferenceRequiresVisualReference},
			} {
				if field.value == nil {
					return fmt.Errorf("video model %q must configure %s", modelName, field.name)
				}
			}
			if *capability.GenerateAudioRequired && !*capability.SupportsGenerateAudio {
				return fmt.Errorf("video model %q cannot require generate_audio when it is unsupported", modelName)
			}
			if *capability.SupportsLastFrame && *capability.LastFrameRequiresFirstFrame && !*capability.SupportsFirstFrame {
				return fmt.Errorf("video model %q cannot require a first frame when first frames are unsupported", modelName)
			}
		}
	}
	return nil
}

func (s ChannelSettings) GetVideoModelCapability(modelName string) (VideoModelCapability, bool) {
	normalizedModel := strings.ToLower(strings.TrimSpace(modelName))
	for configuredModel, capability := range s.VideoModelCapabilities {
		if strings.ToLower(strings.TrimSpace(configuredModel)) != normalizedModel {
			continue
		}
		resolutions := make([]string, 0, len(capability.Resolutions))
		for _, resolution := range capability.Resolutions {
			resolutions = append(resolutions, strings.ToLower(strings.TrimSpace(resolution)))
		}
		ratios := make([]string, 0, len(capability.Ratios))
		for _, ratio := range capability.Ratios {
			ratios = append(ratios, strings.ToLower(strings.TrimSpace(ratio)))
		}
		return VideoModelCapability{
			Resolutions:                           resolutions,
			Ratios:                                ratios,
			MaxReferenceImages:                    capability.MaxReferenceImages,
			MaxReferenceVideos:                    capability.MaxReferenceVideos,
			MaxReferenceAudios:                    capability.MaxReferenceAudios,
			MinDurationSeconds:                    capability.MinDurationSeconds,
			MaxDurationSeconds:                    capability.MaxDurationSeconds,
			SupportsGenerateAudio:                 capability.SupportsGenerateAudio,
			GenerateAudioRequired:                 capability.GenerateAudioRequired,
			SupportsFirstFrame:                    capability.SupportsFirstFrame,
			SupportsLastFrame:                     capability.SupportsLastFrame,
			LastFrameRequiresFirstFrame:           capability.LastFrameRequiresFirstFrame,
			ReferenceImagesIncompatibleWithFrames: capability.ReferenceImagesIncompatibleWithFrames,
			AudioReferenceRequiresVisualReference: capability.AudioReferenceRequiresVisualReference,
		}, true
	}
	return VideoModelCapability{}, false
}

type VertexKeyType string

const (
	VertexKeyTypeJSON   VertexKeyType = "json"
	VertexKeyTypeAPIKey VertexKeyType = "api_key"
)

type AwsKeyType string

const (
	AwsKeyTypeAKSK   AwsKeyType = "ak_sk" // 默认
	AwsKeyTypeApiKey AwsKeyType = "api_key"
)

type ChannelOtherSettings struct {
	AzureResponsesVersion                 string                `json:"azure_responses_version,omitempty"`
	VertexKeyType                         VertexKeyType         `json:"vertex_key_type,omitempty"` // "json" or "api_key"
	OpenRouterEnterprise                  *bool                 `json:"openrouter_enterprise,omitempty"`
	ClaudeBetaQuery                       bool                  `json:"claude_beta_query,omitempty"`          // Claude 渠道是否强制追加 ?beta=true
	AllowServiceTier                      bool                  `json:"allow_service_tier,omitempty"`         // 是否允许 service_tier 透传（默认过滤以避免额外计费）
	AllowInferenceGeo                     bool                  `json:"allow_inference_geo,omitempty"`        // 是否允许 inference_geo 透传（仅 Claude，默认过滤以满足数据驻留合规
	AllowSpeed                            bool                  `json:"allow_speed,omitempty"`                // 是否允许 speed 透传（仅 Claude，默认过滤以避免意外切换推理速度模式）
	AllowSafetyIdentifier                 bool                  `json:"allow_safety_identifier,omitempty"`    // 是否允许 safety_identifier 透传（默认过滤以保护用户隐私）
	DisableStore                          bool                  `json:"disable_store,omitempty"`              // 是否禁用 store 透传（默认允许透传，禁用后可能导致 Codex 无法使用）
	AllowIncludeObfuscation               bool                  `json:"allow_include_obfuscation,omitempty"`  // 是否允许 stream_options.include_obfuscation 透传（默认过滤以避免关闭流混淆保护）
	DisableTaskPollingSleep               bool                  `json:"disable_task_polling_sleep,omitempty"` // 是否跳过异步任务轮询间隔
	AwsKeyType                            AwsKeyType            `json:"aws_key_type,omitempty"`
	UpstreamModelUpdateCheckEnabled       bool                  `json:"upstream_model_update_check_enabled,omitempty"`        // 是否检测上游模型更新
	UpstreamModelUpdateAutoSyncEnabled    bool                  `json:"upstream_model_update_auto_sync_enabled,omitempty"`    // 是否自动同步上游模型更新
	UpstreamModelUpdateLastCheckTime      int64                 `json:"upstream_model_update_last_check_time,omitempty"`      // 上次检测时间
	UpstreamModelUpdateLastDetectedModels []string              `json:"upstream_model_update_last_detected_models,omitempty"` // 上次检测到的可加入模型
	UpstreamModelUpdateLastRemovedModels  []string              `json:"upstream_model_update_last_removed_models,omitempty"`  // 上次检测到的可删除模型
	UpstreamModelUpdateIgnoredModels      []string              `json:"upstream_model_update_ignored_models,omitempty"`       // 手动忽略的模型
	AdvancedCustom                        *AdvancedCustomConfig `json:"advanced_custom,omitempty"`
}

func (s *ChannelOtherSettings) IsOpenRouterEnterprise() bool {
	if s == nil || s.OpenRouterEnterprise == nil {
		return false
	}
	return *s.OpenRouterEnterprise
}

const (
	advancedCustomConverterNone                        = "none"
	advancedCustomConverterClaudeMessagesToOpenAIChat  = "anthropic_messages_to_openai_chat_completions"
	advancedCustomConverterOpenAIChatToClaudeMessages  = "openai_chat_completions_to_anthropic_messages"
	advancedCustomConverterOpenAIChatToOpenAIResponses = "openai_chat_completions_to_openai_responses"
	advancedCustomConverterOpenAIResponsesToOpenAIChat = "openai_responses_to_openai_chat_completions"
	advancedCustomConverterOpenAIResponsesToGemini     = "openai_responses_to_gemini_generate_content"
	advancedCustomConverterGeminiContentToOpenAIChat   = "gemini_generate_content_to_openai_chat_completions"
	advancedCustomConverterOpenAIChatToGeminiContent   = "openai_chat_completions_to_gemini_generate_content"
)

const (
	AdvancedCustomAuthTypeNone   = "none"
	AdvancedCustomAuthTypeHeader = "header"
	AdvancedCustomAuthTypeQuery  = "query"
)

type AdvancedCustomConfig struct {
	Routes []AdvancedCustomRoute `json:"advanced_routes,omitempty"`
}

type AdvancedCustomRoute struct {
	IncomingPath string                   `json:"incoming_path,omitempty"`
	UpstreamPath string                   `json:"upstream_path,omitempty"`
	Converter    string                   `json:"converter,omitempty"`
	Models       []string                 `json:"models,omitempty"`
	Auth         *AdvancedCustomRouteAuth `json:"auth,omitempty"`
}

type AdvancedCustomRouteAuth struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

const (
	advancedCustomModelPlaceholder = "{model}"
	advancedCustomModelRegexPrefix = "re:"
)

const (
	advancedCustomEndpointPathOpenAIChat             = "/v1/chat/completions"
	advancedCustomEndpointPathOpenAIResponses        = "/v1/responses"
	advancedCustomEndpointPathOpenAIResponsesCompact = "/v1/responses/compact"
	advancedCustomEndpointPathClaudeMessages         = "/v1/messages"
	advancedCustomEndpointPathJinaRerank             = "/v1/rerank"
	advancedCustomEndpointPathImageGeneration        = "/v1/images/generations"
	advancedCustomEndpointPathEmbeddings             = "/v1/embeddings"
)

// MatchPath returns the first route whose IncomingPath matches requestPath.
// Matching mirrors the relay adaptor: exact match, {model} placeholder, and
// :generateContent <-> :streamGenerateContent equivalence.
func (c *AdvancedCustomConfig) MatchPath(requestPath string) (AdvancedCustomRoute, bool) {
	if c == nil {
		return AdvancedCustomRoute{}, false
	}
	for _, route := range c.Routes {
		if matchAdvancedCustomIncomingPath(strings.TrimSpace(route.IncomingPath), requestPath) {
			return route, true
		}
	}
	return AdvancedCustomRoute{}, false
}

// MatchPathForModel returns the first route whose IncomingPath and Models match.
// An empty Models list is a catch-all fallback for that incoming path.
func (c *AdvancedCustomConfig) MatchPathForModel(requestPath string, model string) (AdvancedCustomRoute, bool) {
	if c == nil {
		return AdvancedCustomRoute{}, false
	}
	model = strings.TrimSpace(model)
	for _, route := range c.Routes {
		if matchAdvancedCustomIncomingPath(strings.TrimSpace(route.IncomingPath), requestPath) &&
			matchAdvancedCustomRouteModel(route.Models, model) {
			return route, true
		}
	}
	return AdvancedCustomRoute{}, false
}

// SupportsPath reports whether any route matches requestPath.
func (c *AdvancedCustomConfig) SupportsPath(requestPath string) bool {
	_, ok := c.MatchPath(requestPath)
	return ok
}

// SupportsPathForModel reports whether any route matches requestPath and model.
func (c *AdvancedCustomConfig) SupportsPathForModel(requestPath string, model string) bool {
	_, ok := c.MatchPathForModel(requestPath, model)
	return ok
}

func (c *AdvancedCustomConfig) SupportedEndpointTypesForModel(model string) []constant.EndpointType {
	if c == nil {
		return nil
	}
	model = strings.TrimSpace(model)
	endpoints := make([]constant.EndpointType, 0, len(c.Routes))
	seen := make(map[constant.EndpointType]struct{}, len(c.Routes))
	for _, route := range c.Routes {
		if !matchAdvancedCustomRouteModel(route.Models, model) {
			continue
		}
		endpointType, ok := advancedCustomEndpointTypeFromIncomingPath(strings.TrimSpace(route.IncomingPath))
		if !ok {
			continue
		}
		if _, exists := seen[endpointType]; exists {
			continue
		}
		seen[endpointType] = struct{}{}
		endpoints = append(endpoints, endpointType)
	}
	return endpoints
}

func advancedCustomEndpointTypeFromIncomingPath(incomingPath string) (constant.EndpointType, bool) {
	switch incomingPath {
	case advancedCustomEndpointPathOpenAIChat:
		return constant.EndpointTypeOpenAI, true
	case advancedCustomEndpointPathOpenAIResponses:
		return constant.EndpointTypeOpenAIResponse, true
	case advancedCustomEndpointPathOpenAIResponsesCompact:
		return constant.EndpointTypeOpenAIResponseCompact, true
	case advancedCustomEndpointPathClaudeMessages:
		return constant.EndpointTypeAnthropic, true
	case advancedCustomEndpointPathJinaRerank:
		return constant.EndpointTypeJinaRerank, true
	case advancedCustomEndpointPathImageGeneration:
		return constant.EndpointTypeImageGeneration, true
	case advancedCustomEndpointPathEmbeddings:
		return constant.EndpointTypeEmbeddings, true
	default:
		if isAdvancedCustomGeminiIncomingPath(incomingPath) {
			return constant.EndpointTypeGemini, true
		}
		return "", false
	}
}

func isAdvancedCustomGeminiIncomingPath(incomingPath string) bool {
	if !strings.HasPrefix(incomingPath, "/v1beta/models/") {
		return false
	}
	return strings.Contains(incomingPath, ":generateContent") || strings.Contains(incomingPath, ":streamGenerateContent")
}

func matchAdvancedCustomRouteModel(models []string, model string) bool {
	normalizedModels := normalizeAdvancedCustomRouteModels(models)
	if len(normalizedModels) == 0 {
		return true
	}
	for _, allowedModel := range normalizedModels {
		if matchAdvancedCustomRouteModelRule(allowedModel, model) {
			return true
		}
	}
	return false
}

// advancedCustomModelRegexCache caches compiled route model patterns. Route model
// matching runs on the request hot path (distributor affinity, ability filtering,
// channel cache filtering, adaptor resolve), so patterns must not be recompiled per
// request. Invalid patterns are cached as nil to avoid recompiling them as well.
var advancedCustomModelRegexCache sync.Map // pattern string -> *regexp.Regexp (nil when invalid)

func compileAdvancedCustomModelRegex(pattern string) *regexp.Regexp {
	if cached, ok := advancedCustomModelRegexCache.Load(pattern); ok {
		re, _ := cached.(*regexp.Regexp)
		return re
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		re = nil
	}
	advancedCustomModelRegexCache.Store(pattern, re)
	return re
}

func matchAdvancedCustomRouteModelRule(rule string, model string) bool {
	if !strings.HasPrefix(rule, advancedCustomModelRegexPrefix) {
		return rule == model
	}
	pattern := strings.TrimPrefix(rule, advancedCustomModelRegexPrefix)
	if pattern == "" {
		return false
	}
	re := compileAdvancedCustomModelRegex(pattern)
	return re != nil && re.MatchString(model)
}

func matchAdvancedCustomIncomingPath(configuredPath string, requestPath string) bool {
	if matchAdvancedCustomIncomingPathTemplate(configuredPath, requestPath) {
		return true
	}
	if strings.Contains(configuredPath, ":generateContent") {
		streamPath := strings.Replace(configuredPath, ":generateContent", ":streamGenerateContent", 1)
		return matchAdvancedCustomIncomingPathTemplate(streamPath, requestPath)
	}
	return false
}

func matchAdvancedCustomIncomingPathTemplate(configuredPath string, requestPath string) bool {
	if !strings.Contains(configuredPath, advancedCustomModelPlaceholder) {
		return configuredPath == requestPath
	}

	parts := strings.Split(configuredPath, advancedCustomModelPlaceholder)
	if len(parts) != 2 {
		return false
	}
	if !strings.HasPrefix(requestPath, parts[0]) || !strings.HasSuffix(requestPath, parts[1]) {
		return false
	}

	model := strings.TrimSuffix(strings.TrimPrefix(requestPath, parts[0]), parts[1])
	return model != "" && !strings.Contains(model, "/")
}

func IsAdvancedCustomConverterAllowed(converter string) bool {
	switch converter {
	case advancedCustomConverterNone,
		advancedCustomConverterClaudeMessagesToOpenAIChat,
		advancedCustomConverterOpenAIChatToClaudeMessages,
		advancedCustomConverterOpenAIChatToOpenAIResponses,
		advancedCustomConverterOpenAIResponsesToOpenAIChat,
		advancedCustomConverterOpenAIResponsesToGemini,
		advancedCustomConverterGeminiContentToOpenAIChat,
		advancedCustomConverterOpenAIChatToGeminiContent:
		return true
	default:
		return false
	}
}

func (c *AdvancedCustomConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("advanced_custom is required")
	}
	if len(c.Routes) == 0 {
		return fmt.Errorf("advanced_custom requires at least one route")
	}

	paths := make(map[string]*advancedCustomPathModelState, len(c.Routes))
	for i := range c.Routes {
		route := c.Routes[i]
		route.IncomingPath = strings.TrimSpace(route.IncomingPath)
		upstreamPath := strings.TrimSpace(route.UpstreamPath)
		route.Converter = strings.TrimSpace(route.Converter)
		if route.Converter == "" {
			route.Converter = advancedCustomConverterNone
		}

		if route.IncomingPath == "" {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].incoming_path is required", i)
		}
		if !strings.HasPrefix(route.IncomingPath, "/") {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].incoming_path must start with /", i)
		}
		if strings.Contains(route.IncomingPath, "?") {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].incoming_path must not include query", i)
		}
		if err := validateAdvancedCustomRouteModels(i, route.IncomingPath, route.Models, paths); err != nil {
			return err
		}

		if upstreamPath == "" {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].upstream_path is required", i)
		}
		if err := validateAdvancedCustomUpstreamTarget(i, upstreamPath); err != nil {
			return err
		}

		if !IsAdvancedCustomConverterAllowed(route.Converter) {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].converter is not registered: %s", i, route.Converter)
		}
		if err := validateAdvancedCustomConverterPath(i, route.IncomingPath, route.Converter); err != nil {
			return err
		}
		if err := validateAdvancedCustomRouteAuth(i, route.Auth); err != nil {
			return err
		}
	}

	return nil
}

type advancedCustomPathModelState struct {
	catchAllIndex int
	modelIndexes  map[string]int
}

func validateAdvancedCustomRouteModels(index int, incomingPath string, models []string, paths map[string]*advancedCustomPathModelState) error {
	state := paths[incomingPath]
	if state == nil {
		state = &advancedCustomPathModelState{
			catchAllIndex: -1,
			modelIndexes:  make(map[string]int),
		}
		paths[incomingPath] = state
	}

	normalizedModels := normalizeAdvancedCustomRouteModels(models)
	if len(normalizedModels) == 0 {
		if state.catchAllIndex >= 0 {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].models catch-all already exists for incoming_path: %s", index, incomingPath)
		}
		state.catchAllIndex = index
		return nil
	}

	if state.catchAllIndex >= 0 {
		return fmt.Errorf("advanced_custom.advanced_routes[%d].models catch-all route must be last for incoming_path: %s", index, incomingPath)
	}

	seenInRoute := make(map[string]struct{}, len(normalizedModels))
	for _, model := range normalizedModels {
		if err := validateAdvancedCustomRouteModelRule(index, incomingPath, model); err != nil {
			return err
		}
		if _, exists := seenInRoute[model]; exists {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].models contains duplicate model for incoming_path %s: %s", index, incomingPath, model)
		}
		seenInRoute[model] = struct{}{}
		if existingIndex, exists := state.modelIndexes[model]; exists {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].models overlaps with advanced_routes[%d] for incoming_path %s: %s", index, existingIndex, incomingPath, model)
		}
		state.modelIndexes[model] = index
	}
	return nil
}

func validateAdvancedCustomRouteModelRule(index int, incomingPath string, model string) error {
	if !strings.HasPrefix(model, advancedCustomModelRegexPrefix) {
		return nil
	}
	pattern := strings.TrimPrefix(model, advancedCustomModelRegexPrefix)
	if pattern == "" {
		return fmt.Errorf("advanced_custom.advanced_routes[%d].models regex is empty for incoming_path %s: %s", index, incomingPath, model)
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("advanced_custom.advanced_routes[%d].models regex is invalid for incoming_path %s: %s", index, incomingPath, model)
	}
	return nil
}

func normalizeAdvancedCustomRouteModels(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model != "" {
			normalized = append(normalized, model)
		}
	}
	return normalized
}

func validateAdvancedCustomUpstreamTarget(index int, upstreamPath string) error {
	if strings.HasPrefix(upstreamPath, "/") {
		if strings.HasPrefix(upstreamPath, "//") {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].upstream_path must be a full URL or a path starting with /", index)
		}
		return nil
	}

	parsedURL, err := url.Parse(upstreamPath)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("advanced_custom.advanced_routes[%d].upstream_path must be a full URL or a path starting with /", index)
	}
	if !strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https") {
		return fmt.Errorf("advanced_custom.advanced_routes[%d].upstream_path must use http or https", index)
	}
	return nil
}

func validateAdvancedCustomConverterPath(index int, incomingPath string, converter string) error {
	switch converter {
	case advancedCustomConverterNone:
		return nil
	case advancedCustomConverterClaudeMessagesToOpenAIChat:
		if incomingPath == "/v1/messages" {
			return nil
		}
	case advancedCustomConverterOpenAIChatToClaudeMessages,
		advancedCustomConverterOpenAIChatToOpenAIResponses,
		advancedCustomConverterOpenAIChatToGeminiContent:
		if incomingPath == "/v1/chat/completions" {
			return nil
		}
	case advancedCustomConverterOpenAIResponsesToOpenAIChat:
		if incomingPath == "/v1/responses" {
			return nil
		}
	case advancedCustomConverterOpenAIResponsesToGemini:
		if incomingPath == "/v1/responses" {
			return nil
		}
	case advancedCustomConverterGeminiContentToOpenAIChat:
		if strings.Contains(incomingPath, ":generateContent") || strings.Contains(incomingPath, ":streamGenerateContent") {
			return nil
		}
	}
	return fmt.Errorf("advanced_custom.advanced_routes[%d].converter does not match incoming_path: %s", index, converter)
}

func validateAdvancedCustomRouteAuth(index int, auth *AdvancedCustomRouteAuth) error {
	if auth == nil {
		return nil
	}
	authType := strings.TrimSpace(auth.Type)
	switch authType {
	case AdvancedCustomAuthTypeNone:
		return nil
	case AdvancedCustomAuthTypeHeader, AdvancedCustomAuthTypeQuery:
		if strings.TrimSpace(auth.Name) == "" {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].auth.name is required", index)
		}
		if strings.TrimSpace(auth.Value) == "" {
			return fmt.Errorf("advanced_custom.advanced_routes[%d].auth.value is required", index)
		}
		return nil
	default:
		return fmt.Errorf("advanced_custom.advanced_routes[%d].auth.type is invalid: %s", index, auth.Type)
	}
}
