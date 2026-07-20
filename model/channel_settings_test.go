package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelValidateSettingsVideoContentDelivery(t *testing.T) {
	tests := []struct {
		name      string
		delivery  string
		wantError bool
	}{
		{name: "historical empty value defaults to proxy"},
		{name: "proxy", delivery: dto.VideoContentDeliveryProxy},
		{name: "redirect", delivery: dto.VideoContentDeliveryRedirect},
		{name: "invalid", delivery: "automatic", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setting, err := common.Marshal(dto.ChannelSettings{VideoContentDelivery: test.delivery})
			require.NoError(t, err)
			settingText := string(setting)
			channel := &Channel{Setting: &settingText}

			err = channel.ValidateSettings()
			if test.wantError {
				assert.EqualError(t, err, "video_content_delivery must be proxy or redirect")
				return
			}
			require.NoError(t, err)
		})
	}
}
