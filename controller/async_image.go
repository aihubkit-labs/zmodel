package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type asyncImageBillingContext struct {
	PriceData             types.PriceData              `json:"price_data"`
	TieredBillingSnapshot *billingexpr.BillingSnapshot `json:"tiered_billing_snapshot,omitempty"`
	BillingRequestInput   *billingexpr.RequestInput    `json:"billing_request_input,omitempty"`
	TokenGroup            string                       `json:"token_group"`
	UserGroup             string                       `json:"user_group"`
	EstimatedTokens       int                          `json:"estimated_tokens"`
}

func SubmitAsyncImageTask(c *gin.Context) {
	if err := CheckAsyncImageSubmissionInfrastructure(); err != nil {
		respondAsyncImageError(c, http.StatusServiceUnavailable, err.code, err.message)
		return
	}
	requestValue, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAIImage)
	if err != nil {
		respondAsyncImageError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	request, ok := requestValue.(*dto.ImageRequest)
	if !ok {
		respondAsyncImageError(c, http.StatusBadRequest, "invalid_request_error", "invalid image request")
		return
	}
	if request.Stream != nil && *request.Stream {
		respondAsyncImageError(c, http.StatusBadRequest, "async_image_stream_unsupported", "streaming is not supported for asynchronous image generation")
		return
	}
	relayInfo, relayErr := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIImage, request, nil)
	if relayErr != nil {
		respondAsyncImageError(c, http.StatusBadRequest, "invalid_request_error", relayErr.Error())
		return
	}
	meta := request.GetTokenCountMeta()
	if setting.ShouldCheckPromptSensitive() {
		contains, words := service.CheckSensitiveText(meta.CombineText)
		if contains {
			logger.LogWarn(c, fmt.Sprintf("user sensitive words detected: %s", strings.Join(words, ", ")))
			respondAsyncImageError(c, http.StatusBadRequest, string(types.ErrorCodeSensitiveWordsDetected), "sensitive words detected")
			return
		}
	}
	tokens, tokenErr := service.EstimateRequestToken(c, meta, relayInfo)
	if tokenErr != nil {
		respondAsyncImageError(c, http.StatusBadRequest, string(types.ErrorCodeCountTokenFailed), tokenErr.Error())
		return
	}
	relayInfo.SetEstimatePromptTokens(tokens)
	priceData, priceErr := helper.ModelPriceHelper(c, relayInfo, tokens, meta)
	if priceErr != nil {
		respondAsyncImageError(c, http.StatusBadRequest, string(types.ErrorCodeModelPriceError), priceErr.Error())
		return
	}
	relayInfo.ForcePreConsume = true
	bodyStorage, err := common.GetBodyStorage(c)
	if err != nil {
		respondAsyncImageError(c, http.StatusBadRequest, string(types.ErrorCodeReadRequestBodyFailed), err.Error())
		return
	}
	rawBody, err := bodyStorage.Bytes()
	if err != nil {
		respondAsyncImageError(c, http.StatusBadRequest, string(types.ErrorCodeReadRequestBodyFailed), err.Error())
		return
	}
	requestSnapshot, err := asyncImageRequestSnapshot(request)
	if err != nil {
		respondAsyncImageError(c, http.StatusInternalServerError, string(types.ErrorCodeUpdateDataError), err.Error())
		return
	}
	taskID, err := model.GenerateAsyncImageTaskID()
	if err != nil {
		respondAsyncImageError(c, http.StatusInternalServerError, string(types.ErrorCodeUpdateDataError), err.Error())
		return
	}
	billingContextData, err := common.Marshal(asyncImageBillingContext{
		PriceData:             relayInfo.PriceData,
		TieredBillingSnapshot: relayInfo.TieredBillingSnapshot,
		BillingRequestInput:   relayInfo.BillingRequestInput,
		TokenGroup:            relayInfo.TokenGroup,
		UserGroup:             relayInfo.UserGroup,
		EstimatedTokens:       tokens,
	})
	if err != nil {
		respondAsyncImageError(c, http.StatusInternalServerError, string(types.ErrorCodeUpdateDataError), err.Error())
		return
	}
	storageSettings := storage_setting.GetSettings()
	reservedQuota := priceData.QuotaToPreConsume
	if priceData.FreeModel {
		reservedQuota = 0
	}
	tokenReservedQuota := reservedQuota
	if relayInfo.TokenUnlimited {
		tokenReservedQuota = 0
	}
	task := &model.AsyncImageTask{
		TaskID:                 taskID,
		UserID:                 relayInfo.UserId,
		TokenID:                relayInfo.TokenId,
		Status:                 model.AsyncImageStatusQueued,
		OutputAvailability:     model.AsyncImageOutputPending,
		BillingStatus:          model.AsyncImageBillingReserved,
		BillingSource:          service.BillingSourceWallet,
		ReservedQuota:          reservedQuota,
		TokenReservedQuota:     tokenReservedQuota,
		TokenUnlimited:         relayInfo.TokenUnlimited,
		OriginModelName:        relayInfo.OriginModelName,
		UsingGroup:             relayInfo.UsingGroup,
		LastChannelID:          0,
		RequestPayload:         string(rawBody),
		RequestSnapshot:        requestSnapshot,
		BillingContext:         string(billingContextData),
		RetentionSeconds:       storageSettings.RetentionSeconds,
		ArchiveTimeoutSeconds:  storageSettings.ArchiveTimeoutSeconds,
		ArchiveMaxAttempts:     storageSettings.ArchiveMaxAttempts,
		ArchiveRetryDeadlineAt: common.GetTimestamp() + storageSettings.ArchiveRetryWindowSeconds,
		NextAttemptAt:          common.GetTimestamp(),
		SourceKind:             "none",
		AdminNotificationState: "none",
	}
	if err := model.CreateAsyncImageTaskWithReservation(task, relayInfo.UserSetting.BillingPreference); err != nil {
		if errors.Is(err, model.ErrAsyncImageInsufficientQuota) {
			respondAsyncImageError(c, http.StatusForbidden, string(types.ErrorCodeInsufficientUserQuota), "insufficient quota")
			return
		}
		respondAsyncImageError(c, http.StatusInternalServerError, string(types.ErrorCodeUpdateDataError), err.Error())
		return
	}
	if _, _, err := service.EnqueueSystemTask(model.SystemTaskTypeAsyncImageProcess, nil); err != nil {
		logger.LogWarn(c, fmt.Sprintf("failed to wake async image processor: task=%s err=%v", taskID, err))
	}
	c.JSON(http.StatusAccepted, asyncImageTaskResponse(task, nil, nil))
}

// asyncImageRequestSnapshot persists the useful generation parameters while
// excluding image input fields and arbitrary extensions that may contain large
// data URLs or sensitive values.
func asyncImageRequestSnapshot(request *dto.ImageRequest) (string, error) {
	snapshot := map[string]any{
		"model":  request.Model,
		"prompt": request.Prompt,
	}
	if request.N != nil {
		snapshot["n"] = *request.N
	}
	if request.Size != "" {
		snapshot["size"] = request.Size
	}
	if request.Quality != "" {
		snapshot["quality"] = request.Quality
	}
	if request.ResponseFormat != "" {
		snapshot["response_format"] = request.ResponseFormat
	}
	if request.Stream != nil {
		snapshot["stream"] = *request.Stream
	}
	if request.Watermark != nil {
		snapshot["watermark"] = *request.Watermark
	}
	serialized, err := common.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(serialized), nil
}

func GetAsyncImageTask(c *gin.Context) {
	taskID := c.Param("task_id")
	userID := c.GetInt("id")
	task, err := model.GetAsyncImageTaskForUser(taskID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondAsyncImageError(c, http.StatusNotFound, "image_generation_task_not_found", "image generation task not found")
			return
		}
		respondAsyncImageError(c, http.StatusInternalServerError, string(types.ErrorCodeQueryDataError), err.Error())
		return
	}
	now := common.GetTimestamp()
	if task.OutputAvailability == model.AsyncImageOutputAvailable && task.OutputExpiresAt > 0 && task.OutputExpiresAt <= now {
		if err := model.MarkExpiredAsyncImageTask(task.TaskID, now); err != nil {
			respondAsyncImageError(c, http.StatusInternalServerError, string(types.ErrorCodeUpdateDataError), err.Error())
			return
		}
		task.OutputAvailability = model.AsyncImageOutputExpired
	}
	if task.OutputAvailability != model.AsyncImageOutputAvailable {
		c.JSON(http.StatusOK, asyncImageTaskResponse(task, nil, nil))
		return
	}
	objects, err := model.ListStorageObjects(task.TaskID)
	if err != nil || len(objects) == 0 {
		respondAsyncImageError(c, http.StatusServiceUnavailable, "object_storage_temporarily_unavailable", "image objects are temporarily unavailable")
		return
	}
	manifest, err := decodeAsyncImageManifest(task.ArchiveManifest)
	if err != nil || len(manifest) != len(objects) {
		respondAsyncImageError(c, http.StatusServiceUnavailable, "object_storage_temporarily_unavailable", "image manifest is unavailable")
		return
	}
	data := make([]dto.AsyncImageOutputData, 0, len(objects))
	settings := storage_setting.GetSettings()
	for _, object := range objects {
		if object.ExpiresAt <= now {
			if err := model.MarkExpiredAsyncImageTask(task.TaskID, now); err != nil {
				respondAsyncImageError(c, http.StatusInternalServerError, string(types.ErrorCodeUpdateDataError), err.Error())
				return
			}
			task.OutputAvailability = model.AsyncImageOutputExpired
			c.JSON(http.StatusOK, asyncImageTaskResponse(task, nil, nil))
			return
		}
		if object.Status != model.StorageObjectStatusAvailable {
			respondAsyncImageError(c, http.StatusServiceUnavailable, "object_storage_temporarily_unavailable", "image objects are temporarily unavailable")
			return
		}
		storage, err := objectstorage.NewStorage(c.Request.Context(), objectstorage.Config{
			Endpoint:        object.Endpoint,
			Region:          object.Region,
			AccessKey:       settings.AccessKey,
			SecretAccessKey: settings.SecretAccessKey,
		})
		if err != nil {
			respondAsyncImageError(c, http.StatusServiceUnavailable, "object_storage_temporarily_unavailable", "object storage is temporarily unavailable")
			return
		}
		seconds := settings.PresignSeconds
		if remaining := object.ExpiresAt - now; remaining < seconds {
			seconds = remaining
		}
		url, err := storage.PresignGetObject(c.Request.Context(), objectstorage.PresignGetObjectInput{
			Bucket:              object.Bucket,
			Key:                 object.ObjectKey,
			Expires:             time.Duration(seconds) * time.Second,
			ResponseContentType: object.MimeType,
			ResponseDisposition: fmt.Sprintf("inline; filename=\"image-%d.%s\"", object.ObjectIndex, object.Extension),
		})
		if err != nil {
			respondAsyncImageError(c, http.StatusServiceUnavailable, "object_storage_temporarily_unavailable", "failed to sign image URL")
			return
		}
		data = append(data, dto.AsyncImageOutputData{
			Index: object.ObjectIndex,
			URL:   url,
		})
	}
	c.JSON(http.StatusOK, asyncImageTaskResponse(task, data, nil))
}

type asyncImageInfrastructureError struct {
	code    string
	message string
}

func CheckAsyncImageSubmissionInfrastructure() *asyncImageInfrastructureError {
	settings := storage_setting.GetSettings()
	if !settings.Configured() || settings.Validate() != nil {
		return &asyncImageInfrastructureError{code: "object_storage_not_configured", message: "object storage is not configured"}
	}
	if err := service.CheckAsyncImageStaging(); err != nil {
		return &asyncImageInfrastructureError{code: "archive_staging_unavailable", message: "archive staging is unavailable"}
	}
	return nil
}

func asyncImageTaskResponse(task *model.AsyncImageTask, data []dto.AsyncImageOutputData, outputError *dto.AsyncImageTaskError) dto.AsyncImageTaskResponse {
	if data == nil {
		data = []dto.AsyncImageOutputData{}
	}
	response := dto.AsyncImageTaskResponse{
		ID:     task.TaskID,
		Status: task.Status,
		Output: dto.AsyncImageTaskOutput{
			Availability: task.OutputAvailability,
			ExpiresAt:    task.OutputExpiresAt,
			Data:         data,
			Error:        outputError,
		},
	}
	if task.Status == model.AsyncImageStatusFailed {
		response.Error = &dto.AsyncImageTaskError{Code: task.PublicErrorCode, Message: task.PublicErrorMessage}
	} else if task.OutputAvailability == model.AsyncImageOutputFailed {
		response.Output.Error = &dto.AsyncImageTaskError{Code: task.PublicErrorCode, Message: task.PublicErrorMessage}
	}
	return response
}

func decodeAsyncImageManifest(value string) ([]service.AsyncImageManifestItem, error) {
	var manifest []service.AsyncImageManifestItem
	if err := common.UnmarshalJsonStr(value, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func respondAsyncImageError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{"error": gin.H{"type": "invalid_request_error", "code": code, "message": message}})
}
