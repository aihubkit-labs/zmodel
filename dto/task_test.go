package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoParameterErrorDataPreservesExplicitZeroAndFalse(t *testing.T) {
	payload, err := common.Marshal(TaskError{
		Code:    "invalid_generate_audio",
		Message: "generate_audio must be true",
		Data: VideoParameterErrorData{
			Parameter:     "generate_audio",
			Received:      false,
			AllowedValues: []any{true},
			Required:      common.GetPointer(true),
		},
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, common.Unmarshal(payload, &decoded))
	data, ok := decoded["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, data["received"])
	assert.Equal(t, true, data["required"])
	assert.Equal(t, []any{true}, data["allowed_values"])
}

func TestVideoParameterErrorDataOmitsAbsentReceivedValue(t *testing.T) {
	payload, err := common.Marshal(TaskError{
		Code:    "unsupported_parameter",
		Message: "unsupported video parameter",
		Data:    VideoParameterErrorData{Parameter: "aspect_ratio"},
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, common.Unmarshal(payload, &decoded))
	data, ok := decoded["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "aspect_ratio", data["parameter"])
	assert.NotContains(t, data, "received")
}
