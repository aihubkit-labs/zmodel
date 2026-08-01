package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

func TestChannelValidateVideoRequestSettings(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		wantErr string
	}{
		{name: "unconfigured preserves historical behavior", setting: `{}`},
		{name: "OpenAI video", setting: `{"video_protocol":"openai_video"}`},
		{name: "Seedance", setting: `{"video_protocol":"seedance"}`},
		{name: "Agnes Video V2", setting: `{"video_protocol":"agnes_video_v2"}`},
		{name: "invalid protocol", setting: `{"video_protocol":"unsafe"}`, wantErr: "unsupported video_protocol"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &Channel{Setting: common.GetPointer(test.setting)}
			err := channel.ValidateSettings()
			if test.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, test.wantErr)
		})
	}
}
