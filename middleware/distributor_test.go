package middleware

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestAdvancedCustomAsyncImageEditTaskUsesRelayPath(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeAdvancedCustom}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{Routes: []dto.AdvancedCustomRoute{{
			IncomingPath: "/v1/images/edits",
			UpstreamPath: "/v1/images/edits",
			Models:       []string{"gpt-image-1"},
		}}},
	})

	assert.True(t, channelSupportsRequestPath(channel, "/v1/images/edits/tasks", "gpt-image-1"))
	assert.False(t, channelSupportsRequestPath(channel, "/v1/images/generations/tasks", "gpt-image-1"))
}
