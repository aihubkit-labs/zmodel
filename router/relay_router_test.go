package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRelayRouterRegistersAsyncImageEditTaskEndpoints(t *testing.T) {
	engine := gin.New()
	SetRelayRouter(engine)

	routes := make(map[string]bool)
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	assert.True(t, routes[http.MethodPost+" /v1/images/edits/tasks"])
	assert.True(t, routes[http.MethodGet+" /v1/images/edits/tasks/:task_id"])
}
