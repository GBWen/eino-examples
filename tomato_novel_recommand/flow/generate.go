package flow

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Generate wraps single-shot generation.
func Generate(ctx context.Context, llm model.ToolCallingChatModel, in []*schema.Message) *schema.Message {
	result, err := llm.Generate(ctx, in)
	if err != nil {
		log.Fatalf("llm generate failed: %v", err)
	}
	return result
}

// Stream wraps streaming generation.
func Stream(ctx context.Context, llm model.ToolCallingChatModel, in []*schema.Message) *schema.StreamReader[*schema.Message] {
	result, err := llm.Stream(ctx, in)
	if err != nil {
		log.Fatalf("llm generate failed: %v", err)
	}
	return result
}

// ConsumeStream prints streaming tokens to writer and returns the full message.
func ConsumeStream(sr *schema.StreamReader[*schema.Message], w io.Writer) *schema.Message {
	defer sr.Close()

	full := &schema.Message{Role: "assistant"}
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
			fmt.Fprint(w, msg.Content)
			full.Content += msg.Content
		}
	}
	fmt.Fprintln(w)
	return full
}

// StdoutWriter is a helper to write to stdout.
var StdoutWriter io.Writer = os.Stdout
