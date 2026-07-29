package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAsyncImageGenerationRoutesUseResourcePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetRelayRouter(engine)

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	assert.Contains(t, routes, http.MethodPost+" /v1/image-generation-tasks")
	assert.Contains(t, routes, http.MethodGet+" /v1/image-generation-tasks/:task_id")
	assert.NotContains(t, routes, http.MethodPost+" /v1/images/generations/tasks")
	assert.NotContains(t, routes, http.MethodGet+" /v1/images/generations/tasks/:task_id")
	assert.Contains(t, routes, http.MethodPost+" /v1/images/generations")
}
