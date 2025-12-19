/*
 * Tomato novel recommendation demo - simple chat loop entry.
 *
 * Usage (from repo root):
 *   ARK_API_KEY=xxx go run ./tomato_novel_recommand
 *
 * This will start an interactive CLI that asks for your reading preference
 * and returns a list of recommended Chinese web novels each round.
 */

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	log.Printf("=== 番茄小说推荐 Demo ===")
	log.Printf("提示：请先在环境变量中设置 ARK_API_KEY\n")

	// create LLM
	log.Printf("=== 正在创建 Ark Chat 模型 ===\n")
	cm := createArkChatModel(ctx)
	log.Printf("模型创建成功\n\n")

	reader := bufio.NewReader(os.Stdin)
	var history []*schema.Message

	for {
		fmt.Print("请输入你当前想看的小说类型 / 心情 / 偏好（输入 exit 退出）：")
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("read input failed: %v", err)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if strings.EqualFold(text, "exit") {
			fmt.Println("再见，期待下次帮你找书～")
			return
		}

		// build messages based on current preference and chat history
		messages := createMessagesFromTemplate(text, history)

		// streaming generation
		log.Printf("\n=== LLM recommendation result (streaming) ===\n")
		sr := stream(ctx, cm, messages)

		full := &schema.Message{
			Role: "assistant", // ensure role is valid for next round
		}

		for {
			msg, err := sr.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("stream recv failed: %v", err)
			}
			if msg == nil {
				continue
			}
			if msg.Content != "" {
				fmt.Print(msg.Content)
				full.Content += msg.Content
			}
		}
		fmt.Println()

		// update history: user messages + assistant reply
		history = append(history, messages...)
		history = append(history, full)

		fmt.Println("\n----------------------")
	}
}
