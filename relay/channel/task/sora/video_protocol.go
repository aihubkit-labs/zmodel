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

var minimaxH3VideoRequestFields = map[string]struct{}{
	"generate_audio": {}, "watermark": {}, "first_image": {}, "last_image": {},
}

var minimaxH3CommonVideoRequestFields = map[string]struct{}{
	"prompt": {}, "model": {},
	"referenceImages": {}, "referenceVideos": {}, "referenceAudios": {},
	"duration": {}, "resolution": {}, "ratio": {}, "provider_options": {},
}

var minimaxH3VideoFileFields = map[string]struct{}{
	"referenceImageFiles": {}, "referenceVideoFiles": {}, "referenceAudioFiles": {},
	"first_image": {}, "last_image": {},
}

var protectedVideoProviderOptionFields = map[string]struct{}{
	"prompt": {}, "model": {}, "mode": {}, "image": {}, "images": {},
	"referenceimages": {}, "referencevideos": {}, "referenceaudios": {},
	"size": {}, "duration": {}, "seconds": {}, "resolution": {}, "ratio": {},
	"input_reference": {}, "metadata": {}, "provider_options": {},
	"n": {}, "count": {}, "output_count": {}, "outputcount": {}, "outputs": {},
	"num_frames": {}, "numframes": {}, "frame_rate": {}, "framerate": {},
	"width": {}, "height": {},
	"generate_audio": {}, "generateaudio": {}, "watermark": {},
	"first_image": {}, "firstimage": {}, "last_image": {}, "lastimage": {},
	"callback_url": {}, "callbackurl": {}, "webhook": {}, "webhook_url": {}, "webhookurl": {},
	"api_key": {}, "apikey": {}, "authorization": {}, "base_url": {}, "baseurl": {},
}

var nestedSensitiveVideoOptionFields = map[string]struct{}{
	"duration": {}, "seconds": {}, "resolution": {}, "size": {},
	"n": {}, "count": {}, "output_count": {}, "outputcount": {}, "outputs": {},
	"num_frames": {}, "numframes": {}, "frame_rate": {}, "framerate": {},
	"width": {}, "height": {},
	"generate_audio": {}, "generateaudio": {}, "watermark": {},
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
			return videoRequestError(fmt.Sprintf("unsupported video parameter %q", name), "unsupported_parameter")
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
	case dto.VideoProtocolSeedanceMegabyAI:
		return "seedance(megabyai)"
	case dto.VideoProtocolMinimaxH3MegabyAI:
		return "minimax-h3(megabyai)"
	case dto.VideoProtocolAgnesVideoV2:
		return "agnes"
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
	if info.ChannelSetting.VideoProtocol == dto.VideoProtocolMinimaxH3MegabyAI {
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
			"Agnes requests must use referenceImages instead of image, images, or input_reference",
			"invalid_reference_images",
		)
	}
	if len(req.ReferenceImages) > 1 {
		return videoRequestError(
			"Agnes supports at most one reference image",
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
				"Agnes referenceImages must contain a valid HTTP or HTTPS URL that Agnes can access",
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
			fmt.Sprintf("Agnes duration must be between 1 and %d seconds", agnesMaxDurationSeconds),
			"invalid_seconds",
		)
	}
	if strings.TrimSpace(req.Size) != "" {
		return videoRequestError("Agnes requests must use resolution and ratio instead of size", "invalid_size")
	}
	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		return videoRequestError("Agnes resolution is required", "invalid_resolution")
	}
	ratio := strings.TrimSpace(req.Ratio)
	if ratio == "" {
		ratio = agnesDefaultRatio
	}
	if _, _, ok := agnesVideoDimensions(resolution, ratio); !ok {
		if _, supportedResolution := agnesResolutionPixels(resolution); !supportedResolution {
			return videoRequestError("Agnes resolution must use a numeric value ending in p, such as 720p", "invalid_resolution")
		}
		return videoRequestError("Agnes ratio must be one of 16:9, 9:16, 1:1, 4:3, or 3:4", "invalid_ratio")
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

func validateVideoModelCapability(info *relaycommon.RelayInfo, req relaycommon.TaskSubmitReq) *dto.TaskError {
	if info == nil || info.ChannelSetting.VideoProtocol == "" {
		return nil
	}
	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(req.Model)
	}
	capability, ok := info.ChannelSetting.GetVideoModelCapability(modelName)
	if !ok {
		return videoRequestError(
			"video model is not configured for the selected channel",
			"video_model_not_configured",
		)
	}
	if capability.MaxReferenceImages == nil || capability.MaxReferenceVideos == nil || capability.MaxReferenceAudios == nil {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("video model %q has incomplete reference media limits", modelName),
			"video_model_capability_incomplete",
			http.StatusInternalServerError,
		)
	}
	if videoProtocolUsesConfiguredDuration(info.ChannelSetting.VideoProtocol) {
		if capability.MinDurationSeconds == nil || capability.MaxDurationSeconds == nil {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("video model %q has incomplete duration limits", modelName),
				"video_model_capability_incomplete",
				http.StatusInternalServerError,
			)
		}
		if *capability.MinDurationSeconds < 1 ||
			*capability.MaxDurationSeconds > relaycommon.MaxTaskDurationSeconds ||
			*capability.MinDurationSeconds > *capability.MaxDurationSeconds {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("video model %q has invalid duration limits", modelName),
				"video_model_capability_invalid",
				http.StatusInternalServerError,
			)
		}
		info.VideoMinDurationSeconds = *capability.MinDurationSeconds
		info.VideoMaxDurationSeconds = *capability.MaxDurationSeconds
	}
	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		return videoRequestError(fmt.Sprintf("resolution is required for video model %q", modelName), "invalid_resolution")
	}
	resolutionSupported := false
	for _, configuredResolution := range capability.Resolutions {
		if strings.ToLower(strings.TrimSpace(configuredResolution)) == resolution {
			resolutionSupported = true
			break
		}
	}
	if !resolutionSupported {
		return videoRequestError(
			fmt.Sprintf("video model %q does not support resolution %q", modelName, req.Resolution),
			"invalid_resolution",
		)
	}
	info.VideoAllowedResolutions = append([]string(nil), capability.Resolutions...)
	if info.ChannelSetting.VideoProtocol == dto.VideoProtocolMinimaxH3MegabyAI {
		ratio := strings.ToLower(strings.TrimSpace(req.Ratio))
		if ratio == "" {
			return videoRequestError(fmt.Sprintf("ratio is required for video model %q", modelName), "invalid_ratio")
		}
		if !videoValueSupported(capability.Ratios, ratio) {
			return videoRequestError(
				fmt.Sprintf("video model %q does not support ratio %q", modelName, req.Ratio),
				"invalid_ratio",
			)
		}
	}

	for _, media := range []struct {
		name  string
		count int
		limit *int
		code  string
	}{
		{name: "reference images", count: len(req.ReferenceImages) + req.ReferenceImageFiles, limit: capability.MaxReferenceImages, code: "invalid_reference_images"},
		{name: "reference videos", count: len(req.ReferenceVideos) + req.ReferenceVideoFiles, limit: capability.MaxReferenceVideos, code: "invalid_reference_videos"},
		{name: "reference audios", count: len(req.ReferenceAudios) + req.ReferenceAudioFiles, limit: capability.MaxReferenceAudios, code: "invalid_reference_audios"},
	} {
		if media.limit != nil && media.count > *media.limit {
			return videoRequestError(
				fmt.Sprintf("video model %q supports at most %d %s", modelName, *media.limit, media.name),
				media.code,
			)
		}
	}
	if info.ChannelSetting.VideoProtocol == dto.VideoProtocolMinimaxH3MegabyAI {
		return validateMinimaxH3Request(modelName, capability, req)
	}
	return nil
}

func videoProtocolUsesConfiguredDuration(protocol dto.VideoProtocol) bool {
	return protocol == dto.VideoProtocolSeedanceMegabyAI || protocol == dto.VideoProtocolMinimaxH3MegabyAI
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

func validateMinimaxH3Request(modelName string, capability dto.VideoModelCapability, req relaycommon.TaskSubmitReq) *dto.TaskError {
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
				fmt.Errorf("video model %q has incomplete MiniMax capability %s", modelName, field.name),
				"video_model_capability_incomplete",
				http.StatusInternalServerError,
			)
		}
	}
	if len([]rune(req.Prompt)) > 2000 {
		return videoRequestError("prompt cannot exceed 2000 characters for MiniMax H3", "invalid_prompt")
	}
	if !*capability.SupportsGenerateAudio && req.GenerateAudio != nil {
		return videoRequestError("generate_audio is not supported by this video model", "invalid_generate_audio")
	}
	if *capability.GenerateAudioRequired && (req.GenerateAudio == nil || !*req.GenerateAudio) {
		return videoRequestError("generate_audio must be true for this video model", "invalid_generate_audio")
	}

	hasFirstFrame := req.FirstImage != "" || req.FirstImageFile
	hasLastFrame := req.LastImage != "" || req.LastImageFile
	if req.FirstImage != "" && req.FirstImageFile {
		return videoRequestError("first_image must use either a URL or one uploaded file", "invalid_first_image")
	}
	if req.LastImage != "" && req.LastImageFile {
		return videoRequestError("last_image must use either a URL or one uploaded file", "invalid_last_image")
	}
	if req.FirstImage != "" && !isPublicVideoReferenceURL(req.FirstImage) {
		return videoRequestError("first_image must be a valid HTTP or HTTPS URL", "invalid_first_image")
	}
	if req.LastImage != "" && !isPublicVideoReferenceURL(req.LastImage) {
		return videoRequestError("last_image must be a valid HTTP or HTTPS URL", "invalid_last_image")
	}
	if hasFirstFrame && !*capability.SupportsFirstFrame {
		return videoRequestError("first_image is not supported by this video model", "invalid_first_image")
	}
	if hasLastFrame && !*capability.SupportsLastFrame {
		return videoRequestError("last_image is not supported by this video model", "invalid_last_image")
	}
	if hasLastFrame && *capability.LastFrameRequiresFirstFrame && !hasFirstFrame {
		return videoRequestError("last_image requires first_image", "invalid_last_image")
	}
	referenceImageCount := len(req.ReferenceImages) + req.ReferenceImageFiles
	if (hasFirstFrame || hasLastFrame) && referenceImageCount > 0 && *capability.ReferenceImagesIncompatibleWithFrames {
		return videoRequestError("referenceImages cannot be combined with first_image or last_image", "invalid_reference_images")
	}
	referenceAudioCount := len(req.ReferenceAudios) + req.ReferenceAudioFiles
	if referenceAudioCount > 0 && referenceImageCount == 0 && *capability.AudioReferenceRequiresVisualReference {
		return videoRequestError("referenceAudios require at least one referenceImages item", "invalid_reference_audios")
	}
	return nil
}

func isPublicVideoReferenceURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Host != "" && (strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https"))
}

func applyVideoProtocolRequest(protocol dto.VideoProtocol, request relaycommon.TaskSubmitReq, fields map[string]json.RawMessage) error {
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
		return fmt.Errorf("invalid Agnes resolution or ratio")
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
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return videoRequestError(err.Error(), "invalid_request")
	}
	defer form.RemoveAll()
	for name := range form.Value {
		if !videoRequestFieldAllowed(protocol, name, false) {
			return videoRequestError(fmt.Sprintf("unsupported video parameter %q", name), "unsupported_parameter")
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
			return videoRequestError(fmt.Sprintf("unsupported video parameter %q", name), "unsupported_parameter")
		}
		if (name == "first_image" || name == "last_image") && len(form.File[name]) != 1 {
			return videoRequestError(fmt.Sprintf("%s must contain one file", name), "invalid_request")
		}
	}
	return nil
}

func videoRequestFieldAllowed(protocol dto.VideoProtocol, name string, file bool) bool {
	if protocol == dto.VideoProtocolMinimaxH3MegabyAI {
		if file {
			_, ok := minimaxH3VideoFileFields[name]
			return ok
		}
		if _, ok := minimaxH3CommonVideoRequestFields[name]; ok {
			return true
		}
		_, ok := minimaxH3VideoRequestFields[name]
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
				"Agnes reference images require an application/json request with referenceImages containing one HTTP or HTTPS URL",
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
	for name, raw := range options {
		fields[name] = raw
	}
}

func videoRequestError(message, code string) *dto.TaskError {
	return service.TaskErrorWrapperLocal(fmt.Errorf("%s", message), code, http.StatusBadRequest)
}
