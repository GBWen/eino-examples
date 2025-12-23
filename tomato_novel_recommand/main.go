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
	"context"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/service"
)

func main() {
	ctx := context.Background()
	service.Run(ctx)
}
