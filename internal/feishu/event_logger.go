package feishu

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
)

// wsEventLogger logs the event type before the SDK dispatches it, without logging message content.
type wsEventLogger struct {
	base larkcore.Logger
}

func (l wsEventLogger) Debug(_ context.Context, args ...interface{}) {
	if len(args) != 1 {
		return
	}
	message, ok := args[0].(string)
	if !ok || !strings.HasPrefix(message, "receive message, ") {
		return
	}

	_, payload, ok := strings.Cut(message, ", payload: ")
	if !ok {
		log.Println("[Feishu] SDK 收到数据帧，但未找到事件内容")
		return
	}
	var envelope struct {
		Header struct {
			EventType string `json:"event_type"`
		} `json:"header"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil || envelope.Header.EventType == "" {
		log.Println("[Feishu] SDK 收到数据帧，但无法识别事件类型")
		return
	}
	log.Printf("[Feishu] SDK 收到事件: %s", envelope.Header.EventType)
}

func (l wsEventLogger) Info(ctx context.Context, args ...interface{}) {
	l.base.Info(ctx, args...)
}

func (l wsEventLogger) Warn(ctx context.Context, args ...interface{}) {
	l.base.Warn(ctx, args...)
}

func (l wsEventLogger) Error(ctx context.Context, args ...interface{}) {
	l.base.Error(ctx, args...)
}
