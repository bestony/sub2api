package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestWriteOpenAIWSResponsesFailedEvent_EmitsTerminalBeforeClose(t *testing.T) {
	serverErrCh := make(chan error, 1)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionDisabled,
		})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()

		closeOpenAIClientWSWithFailedEvent(
			r.Context(),
			conn,
			coderws.StatusInternalError,
			"upstream websocket proxy failed",
			"write upstream websocket request: write: broken pipe",
		)
		serverErrCh <- nil
	}))
	defer wsServer.Close()

	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	require.NoError(t, err)
	defer func() { _ = clientConn.CloseNow() }()

	readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelRead()
	msgType, payload, readErr := clientConn.Read(readCtx)
	require.NoError(t, readErr)
	require.Equal(t, coderws.MessageText, msgType)
	require.Equal(t, "response.failed", gjson.GetBytes(payload, "type").String())
	require.Equal(t, "failed", gjson.GetBytes(payload, "response.status").String())
	require.Contains(t, gjson.GetBytes(payload, "response.error.message").String(), "broken pipe")

	// Close frame should follow the terminal event.
	closeCtx, cancelClose := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelClose()
	_, _, closeReadErr := clientConn.Read(closeCtx)
	require.Error(t, closeReadErr)

	select {
	case serverErr := <-serverErrCh:
		require.NoError(t, serverErr)
	case <-time.After(3 * time.Second):
		t.Fatal("server handler timed out")
	}
}
