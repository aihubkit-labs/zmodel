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
	agnesDefaultResolution         = "720p"
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

var protectedVideoProviderOptionFields = map[string]struct{}{
	"prompt": {}, "model": {}, "mode": {}, "image": {}, "images": {},
	"referenceimages": {}, "referencevideos": {}, "referenceaudios": {},
	"size": {}, "duration": {}, "seconds": {}, "resolution": {}, "ratio": {},
	"input_reference": {}, "metadata": {}, "provider_options": {},
	"n": {}, "count": {}, "output_count": {}, "outputcount": {}, "outputs": {},
	"num_frames": {}, "numframes": {}, "frame_rate": {}, "framerate": {},
	"width": {}, "height": {},
	"callback_url": {}, "callbackurl": {}, "webhook": {}, "webhook_url": {}, "webhookurl": {},
	"api_key": {}, "apikey": {}, "authorization": {}, "base_url": {}, "baseurl": {},
}

var nestedSensitiveVideoOptionFields = map[string]struct{}{
	"duration": {}, "seconds": {}, "resolution": {}, "size": {},
	"n": {}, "count": {}, "output_count": {}, "outputcount": {}, "outputs": {},
	"num_frames": {}, "numframes": {}, "frame_rate": {}, "framerate": {},
	"width": {}, "height": {},
	"callback_url": {}, "callbackurl": {}, "webhook": {}, "webhook_url": {}, "webhookurl": {},
	"api_key": {}, "apikey": {}, "authorization": {}, "base_url": {}, "baseurl": {},
}

func validateVideoProtocolRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	protocol := info.ChannelSetting.VideoProtocol
	if protocol == "" {
		return nil
	}
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return validateVideoProtocolMultipartFields(c)
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
		if _, ok := commonVideoRequestFields[name]; !ok {
			return videoRequestError(fmt.Sprintf("unsupported video parameter %q for protocol %q", name, protocol), "unsupported_parameter")
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
			return videoRequestError(fmt.Sprintf("provider_options namespace %q does not match video protocol %q", name, protocol), "invalid_provider_options")
		}
	}
	selectedRaw, ok := namespaces[namespace]
	if !ok {
		return videoRequestError(fmt.Sprintf("provider_options.%s is required", namespace), "invalid_provider_options")
	}
	if string(selectedRaw) == "null" {
		return videoRequestError(fmt.Sprintf("provider_options.%s must be an object", namespace), "invalid_provider_options")
	}
	var selected map[string]json.RawMessage
	if err := common.Unmarshal(selectedRaw, &selected); err != nil {
		return videoRequestError(fmt.Sprintf("provider_options.%s must be an object", namespace), "invalid_provider_options")
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
	case dto.VideoProtocolSeedance:
		return "seedance"
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
		resolution = agnesDefaultResolution
	}
	ratio := strings.TrimSpace(req.Ratio)
	if ratio == "" {
		ratio = agnesDefaultRatio
	}
	if _, _, ok := agnesVideoDimensions(resolution, ratio); !ok {
		if _, supportedResolution := agnesResolutionPixels[resolution]; !supportedResolution {
			return videoRequestError("Agnes resolution must be one of 480p, 720p, or 1080p", "invalid_resolution")
		}
		return videoRequestError("Agnes ratio must be one of 16:9, 9:16, 1:1, 4:3, or 3:4", "invalid_ratio")
	}
	if resolution == "1080p" && req.Duration > agnes1080pMaxDurationSeconds {
		return videoRequestError(
			fmt.Sprintf(
				"Agnes 1080p duration must be between 1 and %d seconds to preserve %d fps",
				agnes1080pMaxDurationSeconds,
				agnesPreferredFrameRate,
			),
			"invalid_seconds",
		)
	}
	req.Resolution = resolution
	req.Ratio = ratio
	req.Seconds = ""
	c.Set("task_request", req)
	return nil
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

var agnesResolutionPixels = map[string]int{
	"480p":  480,
	"720p":  720,
	"1080p": 1080,
}

var agnesRatios = map[string][2]int{
	"16:9": {16, 9},
	"9:16": {9, 16},
	"1:1":  {1, 1},
	"4:3":  {4, 3},
	"3:4":  {3, 4},
}

func agnesVideoDimensions(resolution, ratio string) (int, int, bool) {
	pixels, resolutionOK := agnesResolutionPixels[resolution]
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

func validateVideoProtocolMultipartFields(c *gin.Context) *dto.TaskError {
	form, err := c.MultipartForm()
	if err != nil {
		return videoRequestError(err.Error(), "invalid_request")
	}
	for name := range form.Value {
		if _, ok := commonVideoRequestFields[name]; !ok {
			return videoRequestError(fmt.Sprintf("unsupported video parameter %q", name), "unsupported_parameter")
		}
		if name == "provider_options" {
			return videoRequestError("provider_options requires an application/json request", "invalid_provider_options")
		}
	}
	for name := range form.File {
		if _, ok := commonVideoRequestFields[name]; !ok {
			return videoRequestError(fmt.Sprintf("unsupported video parameter %q", name), "unsupported_parameter")
		}
	}
	return nil
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
