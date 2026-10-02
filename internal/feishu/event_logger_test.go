package feishu

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

func TestWSEventLoggerShowsEventTypeWithoutMessageContent(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })

	logger := wsEventLogger{}
	logger.Debug(context.Background(), `receive message, message_type: event, message_id: 123, trace_id: 456, payload: {"header":{"event_type":"im.message.receive_v1"},"event":{"message":{"content":"secret message"}}}`)

	got := output.String()
	if !strings.Contains(got, "SDK 收到事件: im.message.receive_v1") {
		t.Fatalf("event type was not logged: %s", got)
	}
	if strings.Contains(got, "secret message") {
		t.Fatalf("message content was logged: %s", got)
	}
}
