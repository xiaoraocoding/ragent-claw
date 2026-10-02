// cmd/claw/main.go
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ragent-claw/internal/engine"
	"ragent-claw/internal/feishu"
	"ragent-claw/internal/provider"
	"ragent-claw/internal/tools"
)

func main() {
	// 1. 初始化引擎依赖
	workDir, _ := os.Getwd()

	// 默认使用智谱 GLM-4
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
	}
	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")

	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool(workDir))
	registry.Register(tools.NewWriteFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))
	registry.Register(tools.NewEditFileTool(workDir))

	// 开启慢思考
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, true)

	// 2. 通过飞书 SDK 长连接接收事件，无需启动公网 HTTP 回调服务。
	bot := feishu.NewFeishuBot(eng)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("🚀 go-tiny-claw 正在通过飞书 SDK 长连接接收事件")
	if err := bot.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("飞书长连接异常退出: %v", err)
	}
}
