package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestPathForLogUsesExplicitOverride(t *testing.T) {
	tests := []struct {
		name         string
		logPath      string
		expectedPath string
	}{
		{
			name:         "uses actual request path by default",
			expectedPath: "/v1/images/generations",
		},
		{
			name:         "uses external async task path for logs",
			logPath:      "/v1/images/generations/tasks",
			expectedPath: "/v1/images/generations/tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			request, err := http.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			require.NoError(t, err)
			ctx.Request = request
			if tt.logPath != "" {
				common.SetContextKey(ctx, constant.ContextKeyLogRequestPath, tt.logPath)
			}

			assert.Equal(t, tt.expectedPath, RequestPathForLog(ctx))
		})
	}
}
