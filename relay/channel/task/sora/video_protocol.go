package sora

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const (
	videoProviderOptionsContextKey = "video_provider_options"
	agnesDefaultDurationSeconds    = 5
	agnesDefaultRatio              = "16:9"
	agnesPreferredFrameRate        = 24
	agnesMaxNumFrames              = 441
	agnes1080pMaxNumFrames         = 241
	agnesMaxDurationSeconds        = (agnesMaxNumFrames - 1) / agnesPreferredFrameRate
	agnes1080pMaxDurationSeconds   = (agnes1080pMaxNumFrames - 1) / agnesPreferredFrameRate
)

var commonVideoRequestFields = map[string]struct{}{
	"prompt": {}, "model": {}, "mode": {}, "image": {}, "images": {},
	"referenceImages": {}, "referenceVideos": {}, "referenceAudios": {},
	"size": {}, "duration": {}, "seconds": {}, "resolution": {}, "ratio": {},
	"input_reference": {}, "metadata": {}, "provider_options": {},
}

var extendedVideoRequestFields = map[string]struct{}{
	"generate_audio": {}, "watermark": {}, "first_image": {}, "last_image": {}, "seed": {},
}

var lingganyaVideoAliasFields = map[string]struct{}{
	"image": {}, "images": {}, "input_reference": {}, "size": {},
}

var extendedCommonVideoRequestFields = map[string]struct{}{
	"prompt": {}, "model": {},
	"referenceImages": {}, "referenceVideos": {}, "referenceAudios": {},
	"duration": {}, "seconds": {}, "resolution": {}, "ratio": {}, "provider_options": {},
}

var extendedVideoFileFields = map[string]struct{}{
	"referenceImageFiles": {}, "referenceVideoFiles": {}, "referenceAudioFiles": {},
	"first_image": {}, "last_image": {},
}

var protectedVideoProviderOptionFields = map[string]struct{}{
	"prompt": {}, "model": {}, "mode": {}, "image": {}, "images": {},
	"referenceimages": {}, "referencevideos": {}, "referenceaudios": {},
	"reference_images": {}, "reference_videos": {}, "reference_audios": {},
	"size": {}, "duration": {}, "seconds": {}, "resolution": {}, "ratio": {},
	"aspect_ratio":    {},
	"input_reference": {}, "metadata": {}, "provider_options": {},
	"n": {}, "count": {}, "output_count": {}, "outputcount": {}, "outputs": {},
	"num_frames": {}, "numframes": {}, "frame_rate": {}, "framerate": {},
	"width": {}, "height": {},
	"generate_audio": {}, "generateaudio": {}, "watermark": {},
	"seed":        {},
	"first_image": {}, "firstimage": {}, "last_image": {}, "lastimage": {},
	"callback_url": {}, "callbackurl": {}, "webhook": {}, "webhook_url": {}, "webhookurl": {},
	"api_key": {}, "apikey": {}, "authorization": {}, "base_url": {}, "baseurl": {},
}

var nestedSensitiveVideoOptionFields = map[string]struct{}{
	"duration": {}, "seconds": {}, "resolution": {}, "size": {},
	"ratio": {}, "aspect_ratio": {},
	"reference_images": {}, "reference_videos": {}, "reference_audios": {},
	"n": {}, "count": {}, "output_count": {}, "outputcount": {}, "outputs": {},
	"num_frames": {}, "numframes": {}, "frame_rate": {}, "framerate": {},
	"width": {}, "height": {},
	"generate_audio": {}, "generateaudio": {}, "watermark": {},
	"seed":        {},
	"first_image": {}, "firstimage": {}, "last_image": {}, "lastimage": {},
	"callback_url": {}, "callbackurl": {}, "webhook": {}, "webhook_url": {}, "webhookurl": {},
	"api_key": {}, "apikey": {}, "authorization": {}, "base_url": {}, "baseurl": {},
}

func validateVideoProtocolRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	protocol := info.ChannelSetting.VideoProtocol
	if protocol == "" {
		return nil
	}
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return validateVideoProtocolMultipartFields(c, protocol)
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return videoRequestError(err.Error(), "invalid_request")
	}
	body, err := storage.Bytes()
	if err != nil {
		return videoRequestError(err.Error(), "invalid_request")
	}
	var fields map[string]json.RawMessage
	if err := common.Unmarshal(body, &fields); err != nil {
		return videoRequestError(err.Error(), "invalid_json")
	}
	for name := range fields {
		if !videoRequestFieldAllowed(protocol, name, false) {
			return videoParameterError(
				fmt.Sprintf("unsupported video parameter %q", name),
				"unsupported_parameter",
				dto.VideoParameterErrorData{Parameter: name},
			)
		}
	}

	providerOptionsRaw, hasProviderOptions := fields["provider_options"]
	if !hasProviderOptions || string(providerOptionsRaw) == "null" {
		return nil
	}
	var namespaces map[string]json.RawMessage
	if err := common.Unmarshal(providerOptionsRaw, &namespaces); err != nil {
		return videoRequestError("provider_options must be an object", "invalid_provider_options")
	}
	if len(namespaces) == 0 {
		return nil
	}

	namespace := videoProtocolProviderNamespace(protocol)
	for name := range namespaces {
		if name != namespace {
			return videoRequestError(fmt.Sprintf("provider_options contains an unsupported namespace %q", name), "invalid_provider_options")
		}
	}
	selectedRaw, ok := namespaces[namespace]
	if !ok {
		return videoRequestError("provider_options must contain the namespace required by the selected channel", "invalid_provider_options")
	}
	if string(selectedRaw) == "null" {
		return videoRequestError("the selected provider_options namespace must be an object", "invalid_provider_options")
	}
	var selected map[string]json.RawMessage
	if err := common.Unmarshal(selectedRaw, &selected); err != nil {
		return videoRequestError("the selected provider_options namespace must be an object", "invalid_provider_options")
	}
	for name, raw := range selected {
		normalizedName := strings.ToLower(strings.TrimSpace(name))
		if _, commonField := commonVideoRequestFields[name]; commonField {
			return videoRequestError(fmt.Sprintf("provider option %q cannot override a protected parameter", name), "provider_option_conflict")
		}
		if _, protected := protectedVideoProviderOptionFields[normalizedName]; protected {
			return videoRequestError(fmt.Sprintf("provider option %q cannot override a protected parameter", name), "provider_option_conflict")
		}
		if _, exists := fields[name]; exists {
			return videoRequestError(fmt.Sprintf("provider option %q conflicts with a top-level parameter", name), "provider_option_conflict")
		}
		if taskErr := validateNestedVideoProviderOption(raw, name); taskErr != nil {
			return taskErr
		}
	}
	c.Set(videoProviderOptionsContextKey, selected)
	return nil
}

func videoProtocolProviderNamespace(protocol dto.VideoProtocol) string {
	switch protocol {
	case dto.VideoProtocolOpenAI:
		return "openai"
	case dto.VideoProtocolMegabyAI:
		return "megabyai"
	case dto.VideoProtocolGlobalAIOpc:
		return "globalaiopc"
	case dto.VideoProtocolAgnesVideoV2:
		return "agnes"
	case dto.VideoProtocolLingganya:
		return "lingganya"
	default:
		return ""
	}
}

func validateNestedVideoProviderOption(raw json.RawMessage, rootPath string) *dto.TaskError {
	type pendingValue struct {
		raw   json.RawMessage
		path  string
		depth int
	}
	pending := []pendingValue{{raw: raw, path: rootPath, depth: 1}}
	for len(pending) > 0 {
		last := len(pending) - 1
		value := pending[last]
		pending = pending[:last]
		if value.depth > 16 {
			return videoRequestError(fmt.Sprintf("provider option %q exceeds the maximum nesting depth", value.path), "invalid_provider_options")
		}

		switch common.GetJsonType(value.raw) {
		case "object":
			var object map[string]json.RawMessage
			if err := common.Unmarshal(value.raw, &object); err != nil {
				return videoRequestError(fmt.Sprintf("provider option %q contains invalid JSON", value.path), "invalid_provider_options")
			}
			for name, child := range object {
				path := value.path + "." + name
				normalizedName := strings.ToLower(strings.TrimSpace(name))
				if _, protected := nestedSensitiveVideoOptionFields[normalizedName]; protected {
					return videoRequestError(fmt.Sprintf("provider option %q cannot contain protected parameter %q", value.path, path), "provider_option_conflict")
				}
				pending = append(pending, pendingValue{raw: child, path: path, depth: value.depth + 1})
			}
		case "array":
			var array []json.RawMessage
			if err := common.Unmarshal(value.raw, &array); err != nil {
				return videoRequestError(fmt.Sprintf("provider option %q contains invalid JSON", value.path), "invalid_provider_options")
			}
			for index, child := range array {
				pending = append(pending, pendingValue{
					raw: child, path: fmt.Sprintf("%s[%d]", value.path, index), depth: value.depth + 1,
				})
			}
		}
	}
	return nil
}

func normalizeVideoProtocolRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if info.ChannelSetting.VideoProtocol == dto.VideoProtocolLingganya {
		return normalizeLingganyaVideoRequest(c, info)
	}
	if usesExtendedVideoModelCapabilities(info.ChannelSetting.VideoProtocol) {
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return videoRequestError(err.Error(), "invalid_request")
		}
		req.Resolution = strings.ToLower(strings.TrimSpace(req.Resolution))
		req.Ratio = strings.ToLower(strings.TrimSpace(req.Ratio))
		req.FirstImage = strings.TrimSpace(req.FirstImage)
		req.LastImage = strings.TrimSpace(req.LastImage)
		c.Set("task_request", req)
		return nil
	}
	if info.ChannelSetting.VideoProtocol != dto.VideoProtocolAgnesVideoV2 {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return videoRequestError(err.Error(), "invalid_request")
	}
	if strings.TrimSpace(req.Image) != "" || len(req.Images) > 0 || strings.TrimSpace(req.InputReference) != "" {
		return videoRequestError(
			"requests must use referenceImages instead of image, images, or input_reference",
			"invalid_reference_images",
		)
	}
	if len(req.ReferenceImages) > 1 {
		return videoRequestError(
			"this video model supports at most one reference image",
			"invalid_reference_images",
		)
	}
	if len(req.ReferenceImages) == 1 {
		referenceImage := strings.TrimSpace(req.ReferenceImages[0])
		parsedURL, parseErr := url.Parse(referenceImage)
		if parseErr != nil ||
			(!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https")) ||
			parsedURL.Host == "" {
			return videoRequestError(
				"referenceImages must contain a valid HTTP or HTTPS URL that the provider can access",
				"invalid_reference_images",
			)
		}
		req.ReferenceImages[0] = referenceImage
	}
	if req.Seconds != "" {
		seconds, err := strconv.Atoi(req.Seconds)
		if err != nil {
			return videoRequestError("duration must be an integer", "invalid_seconds")
		}
		if req.Duration > 0 && req.Duration != seconds {
			return videoRequestError("duration and seconds must match when both are provided", "duration_conflict")
		}
		if req.Duration == 0 {
			req.Duration = seconds
		}
	}
	if req.Duration == 0 {
		req.Duration = agnesDefaultDurationSeconds
	}
	if req.Duration < 1 || req.Duration > agnesMaxDurationSeconds {
		return videoRequestError(
			fmt.Sprintf("duration must be between 1 and %d seconds", agnesMaxDurationSeconds),
			"invalid_seconds",
		)
	}
	if strings.TrimSpace(req.Size) != "" {
		return videoRequestError("requests must use resolution and ratio instead of size", "invalid_size")
	}
	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		return videoRequestError("resolution is required", "invalid_resolution")
	}
	ratio := strings.TrimSpace(req.Ratio)
	if ratio == "" {
		ratio = agnesDefaultRatio
	}
	if _, _, ok := agnesVideoDimensions(resolution, ratio); !ok {
		if _, supportedResolution := agnesResolutionPixels(resolution); !supportedResolution {
			return videoRequestError("resolution must use a numeric value ending in p, such as 720p", "invalid_resolution")
		}
		return videoRequestError("ratio must be one of 16:9, 9:16, 1:1, 4:3, or 3:4", "invalid_ratio")
	}
	if resolution == "1080p" && req.Duration > agnes1080pMaxDurationSeconds {
		req.Duration = agnes1080pMaxDurationSeconds
	}
	req.Resolution = resolution
	req.Ratio = ratio
	req.Seconds = ""
	c.Set("task_request", req)
	return nil
}

func normalizeLingganyaVideoRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return videoRequestError(err.Error(), "invalid_request")
	}

	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	size := strings.ToLower(strings.TrimSpace(req.Size))
	ratio := strings.ToLower(strings.TrimSpace(req.Ratio))
	if size != "" && ratio != "" && !videoSizesEquivalent(size, ratio) {
		return videoParameterError(
			"size and ratio describe conflicting video dimensions",
			"size_conflict",
			dto.VideoParameterErrorData{Parameter: "ratio", Received: ratio, RelatedParameters: []string{"size", "ratio"}},
		)
	}
	if resolution != "" && (strings.Contains(resolution, "x") || strings.Contains(resolution, ":")) {
		candidate := size
		if candidate == "" {
			candidate = ratio
		}
		if candidate != "" && !videoSizesEquivalent(resolution, candidate) {
			return videoParameterError(
				"resolution and size describe conflicting video dimensions",
				"size_conflict",
				dto.VideoParameterErrorData{Parameter: "size", Received: candidate, RelatedParameters: []string{"resolution", "size", "ratio"}},
			)
		}
		if size == "" {
			size = resolution
		}
		resolution = ""
	}
	if size == "" {
		size = ratio
	}
	capabilityModel := strings.TrimSpace(info.UpstreamModelName)
	if capabilityModel == "" {
		capabilityModel = req.Model
	}
	capability, hasCapability := info.ChannelSetting.GetVideoModelCapability(capabilityModel)
	if resolution == "" && hasCapability && len(capability.Resolutions) > 0 {
		resolution = strings.ToLower(strings.TrimSpace(capability.Resolutions[0]))
	}
	req.Resolution = resolution
	req.Size = size
	if ratio == "" {
		ratio = size
	}
	req.Ratio = ratio

	if req.Seconds != "" {
		seconds, parseErr := strconv.Atoi(req.Seconds)
		if parseErr != nil {
			return videoParameterError("seconds must be an integer", "invalid_seconds", dto.VideoParameterErrorData{Parameter: "seconds", Received: req.Seconds})
		}
		if req.Duration > 0 && req.Duration != seconds {
			return videoParameterError("duration and seconds must match when both are provided", "duration_conflict", dto.VideoParameterErrorData{Parameter: "seconds", Received: req.Seconds, RelatedParameters: []string{"duration"}})
		}
		req.Duration = seconds
	}
	if req.Duration == 0 {
		if hasCapability && capability.DefaultDurationSeconds != nil {
			req.Duration = *capability.DefaultDurationSeconds
		}
	}
	req.Seconds = ""

	references := make([]string, 0, len(req.ReferenceImages)+len(req.Images)+2)
	seenReferences := make(map[string]struct{}, cap(references))
	aliasReferences := append(append([]string(nil), req.ReferenceImages...), req.Images...)
	aliasReferences = append(aliasReferences, req.Image, req.InputReference)
	for _, reference := range aliasReferences {
		reference = strings.TrimSpace(reference)
		if reference == "" {
			continue
		}
		if _, exists := seenReferences[reference]; exists {
			continue
		}
		seenReferences[reference] = struct{}{}
		references = append(references, reference)
	}
	req.ReferenceImages = references
	req.Image = ""
	req.Images = nil
	req.InputReference = ""
	c.Set("task_request", req)
	return nil
}

func videoSizesEquivalent(left, right string) bool {
	leftWidth, leftHeight, leftOK := parseVideoSize(left)
	rightWidth, rightHeight, rightOK := parseVideoSize(right)
	if !leftOK || !rightOK {
		return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
	}
	return leftWidth*rightHeight == rightWidth*leftHeight
}

func parseVideoSize(value string) (int, int, bool) {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(value)), func(r rune) bool {
		return r == 'x' || r == ':'
	})
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, widthErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, heightErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	return width, height, widthErr == nil && heightErr == nil && width > 0 && height > 0
}

func validateVideoModelCapability(info *relaycommon.RelayInfo, req relaycommon.TaskSubmitReq) *dto.TaskError {
	if info == nil || info.ChannelSetting.VideoProtocol == "" {
		return nil
	}
	capabilityModelName := strings.TrimSpace(info.UpstreamModelName)
	if capabilityModelName == "" {
		capabilityModelName = strings.TrimSpace(req.Model)
	}
	publicModelName := strings.TrimSpace(info.OriginModelName)
	if publicModelName == "" {
		publicModelName = strings.TrimSpace(req.Model)
	}
	capability, ok := info.ChannelSetting.GetVideoModelCapability(capabilityModelName)
	if !ok {
		return videoRequestError(
			"video model is not configured for the selected channel",
			"video_model_not_configured",
		)
	}
	if capability.MaxReferenceImages == nil || capability.MaxReferenceVideos == nil || capability.MaxReferenceAudios == nil {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("video model %q has incomplete reference media limits", capabilityModelName),
			"video_model_capability_incomplete",
			http.StatusInternalServerError,
		)
	}
	if usesExtendedVideoModelCapabilities(info.ChannelSetting.VideoProtocol) {
		if capability.SupportsDuration == nil || capability.DurationRequired == nil {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("video model %q has incomplete duration settings", capabilityModelName),
				"video_model_capability_incomplete",
				http.StatusInternalServerError,
			)
		}
		if *capability.SupportsDuration {
			if capability.MinDurationSeconds == nil || capability.MaxDurationSeconds == nil ||
				*capability.MinDurationSeconds < 1 ||
				*capability.MaxDurationSeconds > relaycommon.MaxTaskDurationSeconds ||
				*capability.MinDurationSeconds > *capability.MaxDurationSeconds {
				return service.TaskErrorWrapperLocal(
					fmt.Errorf("video model %q has invalid duration limits", capabilityModelName),
					"video_model_capability_invalid",
					http.StatusInternalServerError,
				)
			}
			info.VideoMinDurationSeconds = *capability.MinDurationSeconds
			info.VideoMaxDurationSeconds = *capability.MaxDurationSeconds
			info.VideoDurationRequired = *capability.DurationRequired
			if len(capability.AllowedDurationSeconds) > 0 {
				duration := req.Duration
				if duration == 0 && strings.TrimSpace(req.Seconds) != "" {
					duration, _ = strconv.Atoi(req.Seconds)
				}
				allowed := false
				allowedValues := make([]string, 0, len(capability.AllowedDurationSeconds))
				for _, allowedDuration := range capability.AllowedDurationSeconds {
					allowedValues = append(allowedValues, strconv.Itoa(allowedDuration))
					if duration == allowedDuration {
						allowed = true
					}
				}
				if duration > 0 && !allowed {
					return videoAllowedValuesError(
						fmt.Sprintf("video model %q does not support duration %d; supported values: %s", publicModelName, duration, strings.Join(allowedValues, ", ")),
						"invalid_seconds", "duration", duration, allowedValues, false,
					)
				}
			}
		} else if req.Duration != 0 || strings.TrimSpace(req.Seconds) != "" {
			return videoParameterError(
				"duration is not supported by this video model",
				"invalid_seconds",
				dto.VideoParameterErrorData{Parameter: "duration", Received: requestDurationValue(req)},
			)
		}
	}
	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	resolutionSupportedByModel := len(capability.Resolutions) > 0
	if !resolutionSupportedByModel && resolution != "" {
		return videoAllowedValuesError(
			fmt.Sprintf("video model %q does not support resolution", publicModelName),
			"invalid_resolution", "resolution", req.Resolution, nil, false,
		)
	}
	if resolutionSupportedByModel && resolution == "" {
		return videoAllowedValuesError(
			fmt.Sprintf("resolution is required for video model %q; supported values: %s", publicModelName, strings.Join(capability.Resolutions, ", ")),
			"invalid_resolution",
			"resolution",
			nil,
			capability.Resolutions,
			true,
		)
	}
	if resolutionSupportedByModel {
		resolutionSupported := false
		for _, configuredResolution := range capability.Resolutions {
			if strings.ToLower(strings.TrimSpace(configuredResolution)) == resolution {
				resolutionSupported = true
				break
			}
		}
		if !resolutionSupported {
			return videoAllowedValuesError(
				fmt.Sprintf("video model %q does not support resolution %q; supported values: %s", publicModelName, req.Resolution, strings.Join(capability.Resolutions, ", ")),
				"invalid_resolution", "resolution", req.Resolution, capability.Resolutions, false,
			)
		}
	}
	info.VideoAllowedResolutions = append([]string(nil), capability.Resolutions...)
	if usesExtendedVideoModelCapabilities(info.ChannelSetting.VideoProtocol) {
		ratio := strings.ToLower(strings.TrimSpace(req.Ratio))
		if capability.RatioRequired == nil {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("video model %q has incomplete ratio settings", capabilityModelName),
				"video_model_capability_incomplete",
				http.StatusInternalServerError,
			)
		}
		if *capability.RatioRequired && ratio == "" {
			return videoAllowedValuesError(
				fmt.Sprintf("ratio is required for video model %q; supported values: %s", publicModelName, strings.Join(capability.Ratios, ", ")),
				"invalid_ratio",
				"ratio",
				nil,
				capability.Ratios,
				true,
			)
		}
		if ratio != "" && !videoValueSupported(capability.Ratios, ratio) {
			return videoAllowedValuesError(
				fmt.Sprintf("video model %q does not support ratio %q; supported values: %s", publicModelName, req.Ratio, strings.Join(capability.Ratios, ", ")),
				"invalid_ratio",
				"ratio",
				req.Ratio,
				capability.Ratios,
				false,
			)
		}
		if ratio != "" && len(capability.SizeMappings) > 0 && resolution != "" {
			mappingKey := resolution + "|" + ratio
			if _, mapped := videoCapabilityMapping(capability.SizeMappings, mappingKey); !mapped {
				return videoParameterError(
					fmt.Sprintf("video model %q does not support resolution %q with ratio %q", publicModelName, resolution, ratio),
					"invalid_ratio",
					dto.VideoParameterErrorData{
						Parameter:         "ratio",
						Received:          req.Ratio,
						RelatedParameters: []string{"resolution", "ratio"},
					},
				)
			}
		}
	}

	referenceImageCount := len(req.ReferenceImages) + req.ReferenceImageFiles
	if capability.FramesAsReferenceImages != nil && *capability.FramesAsReferenceImages {
		if req.FirstImage != "" || req.FirstImageFile {
			referenceImageCount++
		}
		if req.LastImage != "" || req.LastImageFile {
			referenceImageCount++
		}
	}
	for _, media := range []struct {
		name    string
		count   int
		minimum *int
		maximum *int
		code    string
	}{
		{name: "reference images", count: referenceImageCount, minimum: capability.MinReferenceImages, maximum: capability.MaxReferenceImages, code: "invalid_reference_images"},
		{name: "reference videos", count: len(req.ReferenceVideos) + req.ReferenceVideoFiles, minimum: capability.MinReferenceVideos, maximum: capability.MaxReferenceVideos, code: "invalid_reference_videos"},
		{name: "reference audios", count: len(req.ReferenceAudios) + req.ReferenceAudioFiles, minimum: capability.MinReferenceAudios, maximum: capability.MaxReferenceAudios, code: "invalid_reference_audios"},
	} {
		if media.minimum != nil && media.count < *media.minimum {
			return videoIntegerRangeError(
				fmt.Sprintf("video model %q requires %d to %d %s; received %d", publicModelName, *media.minimum, *media.maximum, media.name, media.count),
				media.code,
				strings.ReplaceAll(media.name, " ", "_"),
				int64(media.count),
				int64(*media.minimum),
				int64(*media.maximum),
			)
		}
		if media.maximum != nil && media.count > *media.maximum {
			minimum := int64(0)
			if media.minimum != nil {
				minimum = int64(*media.minimum)
			}
			return videoIntegerRangeError(
				fmt.Sprintf("video model %q supports %d to %d %s; received %d", publicModelName, minimum, *media.maximum, media.name, media.count),
				media.code,
				strings.ReplaceAll(media.name, " ", "_"),
				int64(media.count),
				minimum,
				int64(*media.maximum),
			)
		}
	}
	if usesExtendedVideoModelCapabilities(info.ChannelSetting.VideoProtocol) {
		return validateExtendedVideoRequest(publicModelName, capabilityModelName, capability, req)
	}
	return nil
}

func usesExtendedVideoModelCapabilities(protocol dto.VideoProtocol) bool {
	return protocol == dto.VideoProtocolMegabyAI || protocol == dto.VideoProtocolGlobalAIOpc || protocol == dto.VideoProtocolLingganya
}

func videoValueSupported(configured []string, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range configured {
		if strings.ToLower(strings.TrimSpace(candidate)) == value {
			return true
		}
	}
	return false
}

func validateExtendedVideoRequest(publicModelName, capabilityModelName string, capability dto.VideoModelCapability, req relaycommon.TaskSubmitReq) *dto.TaskError {
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
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("video model %q has incomplete capability %s", capabilityModelName, field.name),
				"video_model_capability_incomplete",
				http.StatusInternalServerError,
			)
		}
	}
	if !*capability.SupportsGenerateAudio && req.GenerateAudio != nil {
		return videoParameterError("generate_audio is not supported by this video model", "invalid_generate_audio", dto.VideoParameterErrorData{Parameter: "generate_audio", Received: *req.GenerateAudio})
	}
	if *capability.GenerateAudioRequired && (req.GenerateAudio == nil || !*req.GenerateAudio) {
		return videoParameterError("generate_audio must be true for this video model; allowed value: true", "invalid_generate_audio", dto.VideoParameterErrorData{Parameter: "generate_audio", Received: false, AllowedValues: []any{true}, Required: common.GetPointer(true)})
	}
	if capability.AssetPreparationMode != "" {
		for _, references := range []struct {
			name   string
			values []string
			code   string
		}{
			{name: "reference_images", values: req.ReferenceImages, code: "invalid_reference_images"},
			{name: "reference_videos", values: req.ReferenceVideos, code: "invalid_reference_videos"},
			{name: "reference_audios", values: req.ReferenceAudios, code: "invalid_reference_audios"},
		} {
			for _, reference := range references.values {
				if !isPublicVideoReferenceURL(reference) {
					return videoParameterError(
						fmt.Sprintf("%s must contain valid HTTP or HTTPS URLs", references.name),
						references.code,
						dto.VideoParameterErrorData{Parameter: references.name, Received: reference, AllowedValues: []any{"HTTP URL", "HTTPS URL"}},
					)
				}
			}
		}
	}

	hasFirstFrame := req.FirstImage != "" || req.FirstImageFile
	hasLastFrame := req.LastImage != "" || req.LastImageFile
	if capability.FramesAsReferenceImages != nil && *capability.FramesAsReferenceImages && capability.MaxReferenceImages != nil {
		combinedImageCount := len(req.ReferenceImages) + req.ReferenceImageFiles
		if hasFirstFrame {
			combinedImageCount++
		}
		if hasLastFrame {
			combinedImageCount++
		}
		if combinedImageCount > *capability.MaxReferenceImages {
			return videoIntegerRangeError(
				fmt.Sprintf("video model %q supports 0 to %d combined reference and frame images; received %d", publicModelName, *capability.MaxReferenceImages, combinedImageCount),
				"invalid_reference_images",
				"combined_reference_and_frame_images",
				int64(combinedImageCount),
				0,
				int64(*capability.MaxReferenceImages),
			)
		}
	}
	if capability.FirstFrameRequired != nil && *capability.FirstFrameRequired && !hasFirstFrame {
		return videoParameterError("first_image is required by this video model", "invalid_first_image", dto.VideoParameterErrorData{Parameter: "first_image", Required: common.GetPointer(true)})
	}
	if capability.LastFrameRequired != nil && *capability.LastFrameRequired && !hasLastFrame {
		return videoParameterError("last_image is required by this video model", "invalid_last_image", dto.VideoParameterErrorData{Parameter: "last_image", Required: common.GetPointer(true)})
	}
	if req.FirstImage != "" && req.FirstImageFile {
		return videoParameterError("first_image must use either a URL or one uploaded file", "invalid_first_image", dto.VideoParameterErrorData{Parameter: "first_image", Received: "URL and uploaded file"})
	}
	if req.LastImage != "" && req.LastImageFile {
		return videoParameterError("last_image must use either a URL or one uploaded file", "invalid_last_image", dto.VideoParameterErrorData{Parameter: "last_image", Received: "URL and uploaded file"})
	}
	if req.FirstImage != "" && !isPublicVideoReferenceURL(req.FirstImage) {
		return videoParameterError("first_image must be a valid HTTP or HTTPS URL", "invalid_first_image", dto.VideoParameterErrorData{Parameter: "first_image", Received: req.FirstImage, AllowedValues: []any{"HTTP URL", "HTTPS URL"}})
	}
	if req.LastImage != "" && !isPublicVideoReferenceURL(req.LastImage) {
		return videoParameterError("last_image must be a valid HTTP or HTTPS URL", "invalid_last_image", dto.VideoParameterErrorData{Parameter: "last_image", Received: req.LastImage, AllowedValues: []any{"HTTP URL", "HTTPS URL"}})
	}
	if hasFirstFrame && !*capability.SupportsFirstFrame {
		return videoParameterError("first_image is not supported by this video model", "invalid_first_image", dto.VideoParameterErrorData{Parameter: "first_image", Received: req.FirstImage})
	}
	if hasLastFrame && !*capability.SupportsLastFrame {
		return videoParameterError("last_image is not supported by this video model", "invalid_last_image", dto.VideoParameterErrorData{Parameter: "last_image", Received: req.LastImage})
	}
	if hasLastFrame && *capability.LastFrameRequiresFirstFrame && !hasFirstFrame {
		return videoParameterError("last_image requires first_image", "invalid_last_image", dto.VideoParameterErrorData{Parameter: "last_image", Received: req.LastImage, RelatedParameters: []string{"first_image"}})
	}
	referenceImageCount := len(req.ReferenceImages) + req.ReferenceImageFiles
	referenceVideoCount := len(req.ReferenceVideos) + req.ReferenceVideoFiles
	if (hasFirstFrame || hasLastFrame) && referenceImageCount > 0 && *capability.ReferenceImagesIncompatibleWithFrames {
		return videoParameterError("referenceImages cannot be combined with first_image or last_image", "invalid_reference_images", dto.VideoParameterErrorData{Parameter: "reference_images", Received: referenceImageCount, RelatedParameters: []string{"first_image", "last_image"}})
	}
	referenceAudioCount := len(req.ReferenceAudios) + req.ReferenceAudioFiles
	if capability.MaxReferenceMediaCount != nil && referenceImageCount+referenceVideoCount+referenceAudioCount > *capability.MaxReferenceMediaCount {
		return videoParameterError("reference media exceeds the maximum total count", "invalid_reference_media", dto.VideoParameterErrorData{Parameter: "reference_media", Received: referenceImageCount + referenceVideoCount + referenceAudioCount, AllowedValues: []any{*capability.MaxReferenceMediaCount}})
	}
	if referenceImageCount == 0 && referenceVideoCount+referenceAudioCount > 0 && capability.ReferenceMediaRequiresVisualReference != nil && *capability.ReferenceMediaRequiresVisualReference {
		return videoParameterError("reference videos and audios require at least one image", "invalid_reference_images", dto.VideoParameterErrorData{Parameter: "reference_images", Received: referenceImageCount, RelatedParameters: []string{"reference_videos", "reference_audios"}})
	}
	if referenceAudioCount > 0 && referenceImageCount+referenceVideoCount == 0 && *capability.AudioReferenceRequiresVisualReference {
		return videoParameterError("referenceAudios require at least one visual reference", "invalid_reference_audios", dto.VideoParameterErrorData{Parameter: "reference_audios", Received: referenceAudioCount, RelatedParameters: []string{"reference_images", "reference_videos"}})
	}
	if (hasFirstFrame || hasLastFrame) && referenceImageCount+referenceVideoCount+referenceAudioCount > 0 && *capability.ReferenceMediaIncompatibleWithFrames {
		return videoParameterError("reference media cannot be combined with first_image or last_image", "invalid_reference_media", dto.VideoParameterErrorData{Parameter: "reference_media", Received: referenceImageCount + referenceVideoCount + referenceAudioCount, RelatedParameters: []string{"first_image", "last_image"}})
	}
	if req.Seed != nil {
		if capability.SupportsSeed != nil && !*capability.SupportsSeed {
			return videoParameterError("seed is not supported by this video model", "invalid_seed", dto.VideoParameterErrorData{Parameter: "seed", Received: *req.Seed})
		}
		if capability.MinSeed != nil && capability.MaxSeed != nil && (*req.Seed < *capability.MinSeed || *req.Seed > *capability.MaxSeed) {
			return videoIntegerRangeError(fmt.Sprintf("seed must be between %d and %d; received %d", *capability.MinSeed, *capability.MaxSeed, *req.Seed), "invalid_seed", "seed", *req.Seed, *capability.MinSeed, *capability.MaxSeed)
		}
	}
	if req.Watermark != nil && capability.SupportsWatermark != nil && !*capability.SupportsWatermark {
		return videoParameterError("watermark is not supported by this video model", "invalid_watermark", dto.VideoParameterErrorData{Parameter: "watermark", Received: *req.Watermark})
	}
	return nil
}

func isPublicVideoReferenceURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Host != "" && (strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https"))
}

func applyVideoProtocolRequest(protocol dto.VideoProtocol, capability dto.VideoModelCapability, request relaycommon.TaskSubmitReq, fields map[string]json.RawMessage) error {
	if protocol == dto.VideoProtocolLingganya {
		for _, name := range []string{
			"duration", "seconds", "resolution", "ratio", "size", "image", "images", "input_reference",
			"referenceImages", "referenceVideos", "referenceAudios", "first_image", "last_image",
			"generate_audio", "watermark", "seed", "provider_options",
		} {
			delete(fields, name)
		}
		mappedFields := map[string]any{"seconds": request.Duration}
		upstreamResolution := strings.TrimSpace(request.Resolution)
		if mappedResolution, ok := videoCapabilityMapping(capability.ResolutionMappings, upstreamResolution); ok {
			upstreamResolution = mappedResolution
		}
		if upstreamResolution != "" {
			mappedFields["resolution"] = upstreamResolution
		}
		publicRatio := strings.TrimSpace(request.Ratio)
		if publicRatio == "" {
			publicRatio = strings.TrimSpace(request.Size)
		}
		upstreamSize := publicRatio
		mappingKey := strings.TrimSpace(request.Resolution) + "|" + publicRatio
		if mappedSize, ok := videoCapabilityMapping(capability.SizeMappings, mappingKey); ok {
			upstreamSize = mappedSize
		}
		if upstreamSize != "" {
			mappedFields["size"] = upstreamSize
		}
		images := append([]string(nil), request.ReferenceImages...)
		if strings.TrimSpace(request.FirstImage) != "" {
			images = append(images, strings.TrimSpace(request.FirstImage))
		}
		if strings.TrimSpace(request.LastImage) != "" {
			images = append(images, strings.TrimSpace(request.LastImage))
		}
		if len(images) > 0 {
			mappedFields["images"] = images
		}
		extra := map[string]any{}
		if len(request.ReferenceVideos) > 0 {
			extra["reference_videos"] = request.ReferenceVideos
		}
		if len(request.ReferenceAudios) > 0 {
			extra["reference_audios"] = request.ReferenceAudios
		}
		if len(extra) > 0 {
			mappedFields["extra"] = extra
		}
		for name, value := range capability.FixedParameters {
			mappedFields[name] = value
		}
		for _, name := range capability.OmitParameters {
			for existingName := range mappedFields {
				if strings.EqualFold(existingName, strings.TrimSpace(name)) {
					delete(mappedFields, existingName)
				}
			}
		}
		for name, value := range mappedFields {
			encoded, err := common.Marshal(value)
			if err != nil {
				return err
			}
			fields[name] = encoded
		}
		return nil
	}
	if usesExtendedVideoModelCapabilities(protocol) {
		globalAIOpcRequest := dto.IsGlobalAIOpcVideoProtocol(protocol)
		if globalAIOpcRequest {
			for _, name := range []string{
				"referenceImages", "referenceVideos", "referenceAudios", "ratio",
				"generate_audio", "watermark", "seed", "size", "seconds",
			} {
				delete(fields, name)
			}
		}
		upstreamResolution := strings.ToLower(strings.TrimSpace(request.Resolution))
		for publicResolution, mappedResolution := range capability.ResolutionMappings {
			if strings.EqualFold(strings.TrimSpace(publicResolution), upstreamResolution) {
				upstreamResolution = strings.TrimSpace(mappedResolution)
				break
			}
		}
		mappedFields := map[string]any{"resolution": upstreamResolution}
		if strings.TrimSpace(request.Ratio) != "" {
			ratioField := "ratio"
			if globalAIOpcRequest {
				ratioField = "aspect_ratio"
			}
			mappedFields[ratioField] = request.Ratio
		} else {
			delete(fields, "ratio")
			delete(fields, "aspect_ratio")
		}
		referenceImages := append([]string(nil), request.ReferenceImages...)
		if capability.FramesAsReferenceImages != nil && *capability.FramesAsReferenceImages {
			delete(fields, "first_image")
			delete(fields, "last_image")
			if strings.TrimSpace(request.FirstImage) != "" {
				referenceImages = append(referenceImages, strings.TrimSpace(request.FirstImage))
			}
			if strings.TrimSpace(request.LastImage) != "" {
				referenceImages = append(referenceImages, strings.TrimSpace(request.LastImage))
			}
		}
		referenceImagesField := "referenceImages"
		referenceVideosField := "referenceVideos"
		referenceAudiosField := "referenceAudios"
		if globalAIOpcRequest {
			referenceImagesField = "reference_images"
			referenceVideosField = "reference_videos"
			referenceAudiosField = "reference_audios"
		}
		if len(referenceImages) > 0 {
			mappedFields[referenceImagesField] = referenceImages
		} else {
			delete(fields, referenceImagesField)
		}
		if len(request.ReferenceVideos) > 0 {
			mappedFields[referenceVideosField] = request.ReferenceVideos
		} else {
			delete(fields, referenceVideosField)
		}
		if len(request.ReferenceAudios) > 0 {
			mappedFields[referenceAudiosField] = request.ReferenceAudios
		} else {
			delete(fields, referenceAudiosField)
		}
		if capability.SupportsGenerateAudio != nil && *capability.SupportsGenerateAudio && request.GenerateAudio != nil {
			mappedFields["generate_audio"] = *request.GenerateAudio
		}
		if capability.SupportsSeed != nil && *capability.SupportsSeed && request.Seed != nil {
			mappedFields["seed"] = *request.Seed
		}
		if capability.SupportsWatermark != nil && *capability.SupportsWatermark && request.Watermark != nil {
			mappedFields["watermark"] = *request.Watermark
		}
		if capability.AutoReferenceMode != nil && *capability.AutoReferenceMode {
			hasFrames := strings.TrimSpace(request.FirstImage) != "" || strings.TrimSpace(request.LastImage) != ""
			referenceMode := capability.ReferenceModeForReferences
			if hasFrames {
				referenceMode = capability.ReferenceModeForFrames
			}
			mappedFields["reference_mode"] = strings.TrimSpace(referenceMode)
		}
		for name, value := range capability.FixedParameters {
			mappedFields[name] = value
		}
		for _, name := range capability.OmitParameters {
			normalizedName := strings.TrimSpace(name)
			for existingName := range fields {
				if strings.EqualFold(existingName, normalizedName) {
					delete(fields, existingName)
				}
			}
			for existingName := range mappedFields {
				if strings.EqualFold(existingName, normalizedName) {
					delete(mappedFields, existingName)
				}
			}
		}
		for name, value := range mappedFields {
			encoded, err := common.Marshal(value)
			if err != nil {
				return err
			}
			fields[name] = encoded
		}
		return nil
	}
	if protocol != dto.VideoProtocolAgnesVideoV2 {
		return nil
	}
	delete(fields, "duration")
	delete(fields, "seconds")
	delete(fields, "num_frames")
	delete(fields, "frame_rate")
	delete(fields, "resolution")
	delete(fields, "ratio")
	delete(fields, "size")
	delete(fields, "width")
	delete(fields, "height")
	delete(fields, "image")
	delete(fields, "images")
	delete(fields, "referenceImages")
	delete(fields, "input_reference")
	if request.Duration <= 0 {
		return nil
	}
	if len(request.ReferenceImages) == 1 {
		imageJSON, err := common.Marshal(request.ReferenceImages[0])
		if err != nil {
			return err
		}
		fields["image"] = imageJSON
	}
	width, height, ok := agnesVideoDimensions(request.Resolution, request.Ratio)
	if !ok {
		return fmt.Errorf("invalid resolution or ratio")
	}
	numFrames, frameRate := agnesVideoFrameParameters(request.Duration)
	numFramesJSON, err := common.Marshal(numFrames)
	if err != nil {
		return err
	}
	frameRateJSON, err := common.Marshal(frameRate)
	if err != nil {
		return err
	}
	widthJSON, err := common.Marshal(width)
	if err != nil {
		return err
	}
	heightJSON, err := common.Marshal(height)
	if err != nil {
		return err
	}
	fields["num_frames"] = numFramesJSON
	fields["frame_rate"] = frameRateJSON
	fields["width"] = widthJSON
	fields["height"] = heightJSON
	return nil
}

func videoCapabilityMapping(mappings map[string]string, publicValue string) (string, bool) {
	for configuredValue, upstreamValue := range mappings {
		if strings.EqualFold(strings.TrimSpace(configuredValue), strings.TrimSpace(publicValue)) {
			return strings.TrimSpace(upstreamValue), true
		}
	}
	return "", false
}

func agnesVideoFrameParameters(duration int) (int, int) {
	return duration*agnesPreferredFrameRate + 1, agnesPreferredFrameRate
}

var agnesRatios = map[string][2]int{
	"16:9": {16, 9},
	"9:16": {9, 16},
	"1:1":  {1, 1},
	"4:3":  {4, 3},
	"3:4":  {3, 4},
}

func agnesVideoDimensions(resolution, ratio string) (int, int, bool) {
	pixels, resolutionOK := agnesResolutionPixels(resolution)
	ratioParts, ratioOK := agnesRatios[ratio]
	if !resolutionOK || !ratioOK {
		return 0, 0, false
	}
	numerator, denominator := ratioParts[0], ratioParts[1]
	if numerator >= denominator {
		width := (pixels*numerator + denominator/2) / denominator
		if width%2 != 0 {
			width++
		}
		return width, pixels, true
	}
	height := (pixels*denominator + numerator/2) / numerator
	if height%2 != 0 {
		height++
	}
	return pixels, height, true
}

func agnesResolutionPixels(resolution string) (int, bool) {
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	if !strings.HasSuffix(resolution, "p") {
		return 0, false
	}
	pixels, err := strconv.Atoi(strings.TrimSuffix(resolution, "p"))
	if err != nil || pixels <= 0 || pixels > 4320 {
		return 0, false
	}
	return pixels, true
}

func validateVideoProtocolMultipartFields(c *gin.Context, protocol dto.VideoProtocol) *dto.TaskError {
	if dto.IsGlobalAIOpcVideoProtocol(protocol) || protocol == dto.VideoProtocolLingganya {
		return videoRequestError("this video model requires an application/json request with public media URLs", "unsupported_content_type")
	}
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return videoRequestError(err.Error(), "invalid_request")
	}
	defer form.RemoveAll()
	for name := range form.Value {
		if !videoRequestFieldAllowed(protocol, name, false) {
			return videoParameterError(fmt.Sprintf("unsupported video parameter %q", name), "unsupported_parameter", dto.VideoParameterErrorData{Parameter: name})
		}
		if (name == "first_image" || name == "last_image") && len(form.Value[name]) != 1 {
			return videoRequestError(fmt.Sprintf("%s must be provided once", name), "invalid_request")
		}
		if name == "provider_options" {
			return videoRequestError("provider_options requires an application/json request", "invalid_provider_options")
		}
	}
	for name := range form.File {
		if !videoRequestFieldAllowed(protocol, name, true) {
			return videoParameterError(fmt.Sprintf("unsupported video parameter %q", name), "unsupported_parameter", dto.VideoParameterErrorData{Parameter: name})
		}
		if (name == "first_image" || name == "last_image") && len(form.File[name]) != 1 {
			return videoRequestError(fmt.Sprintf("%s must contain one file", name), "invalid_request")
		}
	}
	return nil
}

func videoRequestFieldAllowed(protocol dto.VideoProtocol, name string, file bool) bool {
	if usesExtendedVideoModelCapabilities(protocol) {
		if file {
			_, ok := extendedVideoFileFields[name]
			return ok
		}
		if _, ok := extendedCommonVideoRequestFields[name]; ok {
			return true
		}
		if protocol == dto.VideoProtocolLingganya {
			_, ok := lingganyaVideoAliasFields[name]
			if ok {
				return true
			}
		}
		_, ok := extendedVideoRequestFields[name]
		return ok
	}
	if _, ok := commonVideoRequestFields[name]; ok {
		return true
	}
	return false
}

func validateAgnesMultipartReferenceInput(c *gin.Context) *dto.TaskError {
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return videoRequestError(err.Error(), "invalid_request")
	}
	defer form.RemoveAll()
	for _, name := range []string{"referenceImages", "image", "images", "input_reference"} {
		if len(form.Value[name]) > 0 || len(form.File[name]) > 0 {
			return videoRequestError(
				"reference images require an application/json request with referenceImages containing one HTTP or HTTPS URL",
				"invalid_reference_images",
			)
		}
	}
	return nil
}

func mergeVideoProviderOptions(c *gin.Context, fields map[string]json.RawMessage, protocol dto.VideoProtocol) {
	if protocol == "" {
		return
	}
	delete(fields, "provider_options")
	value, ok := c.Get(videoProviderOptionsContextKey)
	if !ok {
		return
	}
	options, ok := value.(map[string]json.RawMessage)
	if !ok {
		return
	}
	if protocol == dto.VideoProtocolLingganya {
		extra := make(map[string]json.RawMessage, len(options))
		if existing, exists := fields["extra"]; exists {
			_ = common.Unmarshal(existing, &extra)
		}
		for name, raw := range options {
			extra[name] = raw
		}
		encoded, err := common.Marshal(extra)
		if err == nil {
			fields["extra"] = encoded
		}
		return
	}
	for name, raw := range options {
		fields[name] = raw
	}
}

func videoRequestError(message, code string) *dto.TaskError {
	return service.TaskErrorWrapperLocal(fmt.Errorf("%s", message), code, http.StatusBadRequest)
}

func videoParameterError(message, code string, data dto.VideoParameterErrorData) *dto.TaskError {
	taskErr := videoRequestError(message, code)
	taskErr.Data = data
	return taskErr
}

func videoAllowedValuesError(message, code, parameter string, received any, allowedValues []string, required bool) *dto.TaskError {
	values := make([]any, 0, len(allowedValues))
	for _, value := range allowedValues {
		values = append(values, value)
	}
	data := dto.VideoParameterErrorData{
		Parameter:     parameter,
		Received:      received,
		AllowedValues: values,
	}
	if required {
		data.Required = common.GetPointer(true)
	}
	return videoParameterError(message, code, data)
}

func videoIntegerRangeError(message, code, parameter string, received, minimum, maximum int64) *dto.TaskError {
	return videoParameterError(message, code, dto.VideoParameterErrorData{
		Parameter: parameter,
		Received:  received,
		Minimum:   common.GetPointer(minimum),
		Maximum:   common.GetPointer(maximum),
	})
}

func requestDurationValue(req relaycommon.TaskSubmitReq) any {
	if strings.TrimSpace(req.Seconds) != "" {
		return req.Seconds
	}
	return req.Duration
}
