package controller

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsyncImageEditRelayRequestReplaysMultipartInput(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	require.NoError(t, form.WriteField("model", "gpt-image-1"))
	require.NoError(t, form.WriteField("prompt", "make the sky blue"))
	require.NoError(t, form.WriteField("n", "2"))
	imagePart, err := form.CreateFormFile("image", "input.png")
	require.NoError(t, err)
	inputImage := []byte{'P', 'N', 'G', 0, 1, 2, 3}
	_, err = imagePart.Write(inputImage)
	require.NoError(t, err)
	require.NoError(t, form.Close())

	task := &model.AsyncImageTask{
		RequestPath:        "/v1/images/edits",
		RequestContentType: form.FormDataContentType(),
		RequestBody:        append([]byte(nil), body.Bytes()...),
	}
	request, err := newAsyncImageRelayRequest(task)
	require.NoError(t, err)
	assert.Equal(t, "/v1/images/edits", request.URL.Path)
	assert.Equal(t, task.RequestContentType, request.Header.Get("Content-Type"))

	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginContext.Request = request
	t.Cleanup(func() {
		if ginContext.Request.MultipartForm != nil {
			_ = ginContext.Request.MultipartForm.RemoveAll()
		}
		common.CleanupBodyStorage(ginContext)
	})
	requestValue, err := helper.GetAndValidateRequest(ginContext, types.RelayFormatOpenAIImage)
	require.NoError(t, err)
	imageRequest, ok := requestValue.(*dto.ImageRequest)
	require.True(t, ok)
	assert.Equal(t, "gpt-image-1", imageRequest.Model)
	assert.Equal(t, "make the sky blue", imageRequest.Prompt)
	require.NotNil(t, imageRequest.N)
	assert.Equal(t, uint(2), *imageRequest.N)

	require.NotNil(t, ginContext.Request.MultipartForm)
	require.Len(t, ginContext.Request.MultipartForm.File["image"], 1)
	file, err := ginContext.Request.MultipartForm.File["image"][0].Open()
	require.NoError(t, err)
	defer file.Close()
	replayedImage, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, inputImage, replayedImage)
}

func TestAsyncImageRelayRequestSupportsLegacyGenerationPayload(t *testing.T) {
	task := &model.AsyncImageTask{RequestPayload: `{"model":"legacy-image","prompt":"red apple"}`}

	request, err := newAsyncImageRelayRequest(task)
	require.NoError(t, err)
	assert.Equal(t, "/v1/images/generations", request.URL.Path)
	assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	assert.JSONEq(t, task.RequestPayload, string(body))
}
