package controller

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func ListVideoModelCapabilityTemplates(c *gin.Context) {
	protocol := dto.VideoProtocol(strings.TrimSpace(c.Query("video_protocol")))
	if protocol != "" && protocol != dto.VideoProtocolGlobalAIOpc && protocol != dto.VideoProtocolMegabyAI && protocol != dto.VideoProtocolLingganya {
		common.ApiErrorMsg(c, "unsupported video protocol")
		return
	}
	templates, err := model.ListVideoModelCapabilityTemplates(protocol)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]dto.VideoModelCapabilityTemplate, 0, len(templates))
	for index := range templates {
		item, err := templates[index].ToDTO()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		items = append(items, item)
	}
	common.ApiSuccess(c, items)
}

func SaveVideoModelCapabilityTemplate(c *gin.Context) {
	var request dto.VideoModelCapabilityTemplate
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}
	request.ModelID = strings.TrimSpace(request.ModelID)
	request.Name = strings.TrimSpace(request.Name)
	if request.VideoProtocol != dto.VideoProtocolGlobalAIOpc && request.VideoProtocol != dto.VideoProtocolMegabyAI && request.VideoProtocol != dto.VideoProtocolLingganya {
		common.ApiErrorMsg(c, "unsupported video protocol")
		return
	}
	if request.ModelID == "" {
		common.ApiErrorMsg(c, "model_id is required")
		return
	}
	if len(request.ModelID) > 128 {
		common.ApiErrorMsg(c, "model_id cannot exceed 128 characters")
		return
	}
	if request.Name == "" {
		request.Name = request.ModelID
	}
	if len(request.Name) > 256 {
		common.ApiErrorMsg(c, "name cannot exceed 256 characters")
		return
	}
	request.SourceURL = strings.TrimSpace(request.SourceURL)
	if request.SourceURL != "" {
		parsedURL, err := url.ParseRequestURI(request.SourceURL)
		if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			common.ApiErrorMsg(c, "source_url must be an HTTP or HTTPS URL")
			return
		}
	}
	settings := dto.ChannelSettings{
		VideoProtocol: request.VideoProtocol,
		VideoModelCapabilities: map[string]dto.VideoModelCapability{
			request.ModelID: request.Capability,
		},
	}
	if err := settings.ValidateVideoRequestSettings(); err != nil {
		common.ApiError(c, fmt.Errorf("invalid video model capability: %w", err))
		return
	}
	capabilityJSON, err := common.Marshal(request.Capability)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	template := model.VideoModelCapabilityTemplate{
		ID:             request.ID,
		VideoProtocol:  request.VideoProtocol,
		ModelID:        request.ModelID,
		Name:           request.Name,
		CapabilityJSON: string(capabilityJSON),
		Source:         "manual",
		SourceURL:      request.SourceURL,
	}
	if err := model.SaveVideoModelCapabilityTemplate(&template); err != nil {
		common.ApiError(c, err)
		return
	}
	item, err := template.ToDTO()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func DeleteVideoModelCapabilityTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid video model capability template ID")
		return
	}
	if err := model.DeleteVideoModelCapabilityTemplate(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
