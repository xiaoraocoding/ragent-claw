package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"ragent-claw/internal/engine"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

type FeishuBot struct {
	client    *lark.Client
	appID     string
	appSecret string
	engine    *engine.AgentEngine
}

func NewFeishuBot(eng *engine.AgentEngine) *FeishuBot {
	appID := os.Getenv("FEISHU_APP_ID")
	appSecret := os.Getenv("FEISHU_APP_SECRET")

	if appID == "" || appSecret == "" {
		log.Fatal("请设置 FEISHU_APP_ID 和 FEISHU_APP_SECRET")
	}

	client := lark.NewClient(appID, appSecret)

	return &FeishuBot{
		client:    client,
		appID:     appID,
		appSecret: appSecret,
		engine:    eng,
	}
}

func (b *FeishuBot) GetEventDispatcher() *dispatcher.EventDispatcher {
	handler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			log.Println("[Feishu] im.message.receive_v1 事件已到达")
			if event == nil || event.Event == nil || event.Event.Message == nil {
				log.Printf("[Feishu] 收到的消息事件缺少消息内容，已忽略")
				return nil
			}

			message := event.Event.Message
			if message.MessageType == nil || *message.MessageType != larkim.MsgTypeText {
				log.Printf("[Feishu] 暂不处理非文本消息")
				return nil
			}
			if message.ChatId == nil || message.Content == nil {
				log.Printf("[Feishu] 文本消息缺少会话 ID 或内容，已忽略")
				return nil
			}

			var content struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal([]byte(*message.Content), &content); err != nil {
				log.Printf("[Feishu] 解析文本消息失败: %v", err)
				return nil
			}
			if strings.TrimSpace(content.Text) == "" {
				log.Printf("[Feishu] 收到空文本消息，已忽略")
				return nil
			}

			chatID := *message.ChatId
			log.Printf("[Feishu] 收到文本消息，会话 %s，正在调用 Agent", chatID)

			go b.handleAgentRun(chatID, content.Text)

			return nil
		}).
		OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
			// 消息已读事件，静默忽略
			return nil
		}).
		OnP2ChatAccessEventBotP2pChatEnteredV1(func(ctx context.Context, _ *larkim.P2ChatAccessEventBotP2pChatEnteredV1) error {
			log.Println("[Feishu] 收到进入机器人单聊事件；此事件不会触发 Agent，需收到 im.message.receive_v1")
			return nil
		})

	return handler
}

func (b *FeishuBot) Start(ctx context.Context) error {
	wsClient := larkws.NewClient(
		b.appID,
		b.appSecret,
		larkws.WithEventHandler(b.GetEventDispatcher()),
		larkws.WithAutoReconnect(true),
		larkws.WithLogLevel(larkcore.LogLevelDebug),
		larkws.WithLogger(wsEventLogger{base: larkcore.NewDefaultLogger(larkcore.LogLevelInfo)}),
		larkws.WithOnReady(func() {
			log.Println("[Feishu] SDK 长连接已建立，等待 im.message.receive_v1 消息事件")
		}),
		larkws.WithOnReconnecting(func() {
			log.Println("[Feishu] SDK 长连接中断，正在重连")
		}),
		larkws.WithOnReconnected(func() {
			log.Println("[Feishu] SDK 长连接已恢复")
		}),
		larkws.WithOnError(func(err error) {
			log.Printf("[Feishu] SDK 长连接错误: %v\n", err)
		}),
	)

	return wsClient.Start(ctx)
}

func (b *FeishuBot) handleAgentRun(chatId string, prompt string) {
	reporter := &FeishuReporter{
		client: b.client,
		chatId: chatId,
	}

	err := b.engine.Run(context.Background(), prompt, reporter)
	if err != nil {
		reporter.sendMsg(fmt.Sprintf("❌ Agent 运行崩溃: %v", err))
	}
}

type FeishuReporter struct {
	client *lark.Client
	chatId string
}

func (r *FeishuReporter) sendMsg(text string) {
	// Build text message content
	textContent := map[string]string{
		"text": text,
	}
	contentBytes, _ := json.Marshal(textContent)
	contentStr := string(contentBytes)

	msgReq := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.CreateMessageV1ReceiveIDTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(r.chatId).
			MsgType(larkim.MsgTypeText).
			Content(contentStr).
			Build()).
		Build()

	resp, err := r.client.Im.Message.Create(context.Background(), msgReq)
	if err != nil {
		log.Printf("[Feishu] 发送回复失败: %v", err)
		return
	}
	if resp == nil {
		log.Printf("[Feishu] 发送回复失败: 飞书返回了空响应")
		return
	}
	if !resp.Success() {
		log.Printf("[Feishu] 发送回复失败: code=%d, message=%s", resp.Code, resp.Msg)
		return
	}
	log.Printf("[Feishu] 回复已发送，会话 %s", r.chatId)
}

func (r *FeishuReporter) OnThinking(ctx context.Context) {
	r.sendMsg("🤔 模型正在慢思考 (Thinking)...")
}

func (r *FeishuReporter) OnToolCall(ctx context.Context, toolName string, args string) {
	r.sendMsg(fmt.Sprintf("🛠️ **正在执行工具**：`%s`\n参数：`%s`", toolName, args))
}

func (r *FeishuReporter) OnToolResult(ctx context.Context, toolName string, result string, isError bool) {
	if isError {
		r.sendMsg(fmt.Sprintf("⚠️ **执行报错** (%s)：\n%s", toolName, result))
	} else {
		r.sendMsg(fmt.Sprintf("✅ **执行成功** (%s)", toolName))
	}
}

func (r *FeishuReporter) OnMessage(ctx context.Context, content string) {
	r.sendMsg(content)
}

var _ engine.Reporter = (*FeishuReporter)(nil)
