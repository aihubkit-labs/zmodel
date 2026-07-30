package taskcommon

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
)

func TestBuildProxyURLFallsBackToServerAddress(t *testing.T) {
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://frontend.example.com/"

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalAPIServerAddress, hadAPIServerAddress := common.OptionMap["ApiServerAddress"]
	common.OptionMap["ApiServerAddress"] = ""
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		system_setting.ServerAddress = originalServerAddress
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadAPIServerAddress {
			common.OptionMap["ApiServerAddress"] = originalAPIServerAddress
		} else {
			delete(common.OptionMap, "ApiServerAddress")
		}
	})

	assert.Equal(t, "https://frontend.example.com/v1/videos/task_public/content", BuildProxyURL("task_public"))
}
