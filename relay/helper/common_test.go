package helper

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type progressDeadlineWriter struct {
	*httptest.ResponseRecorder
	writes    []int
	deadlines []time.Time
}

func (w *progressDeadlineWriter) SetWriteDeadline(deadline time.Time) error {
	w.deadlines = append(w.deadlines, deadline)
	return nil
}

func (w *progressDeadlineWriter) Write(p []byte) (int, error) {
	w.writes = append(w.writes, len(p))
	return w.ResponseRecorder.Write(p)
}

func (w *progressDeadlineWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *progressDeadlineWriter) CloseNotify() <-chan bool {
	return make(chan bool)
}

func (w *progressDeadlineWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("hijack not supported")
}

type shortStreamWriter struct {
	*progressDeadlineWriter
}

func (w *shortStreamWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func TestStringDataWritesLargeEventInProgressChunks(t *testing.T) {
	writer := &progressDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	c, _ := gin.CreateTestContext(writer)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	payload := strings.Repeat("a", streamWriteChunkSize*3+17)

	err := StringData(c, payload)

	require.NoError(t, err)
	assert.Equal(t, "data: "+payload+"\n\n", writer.Body.String())
	assert.GreaterOrEqual(t, len(writer.deadlines), 5, "deadline must refresh as a large event makes progress")
	for _, size := range writer.writes {
		assert.LessOrEqual(t, size, streamWriteChunkSize)
	}
}

func TestStringDataReturnsShortWrite(t *testing.T) {
	writer := &shortStreamWriter{progressDeadlineWriter: &progressDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}}
	c, _ := gin.CreateTestContext(writer)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	err := StringData(c, "image-data")

	require.Error(t, err)
	assert.ErrorIs(t, err, io.ErrShortWrite)
}
