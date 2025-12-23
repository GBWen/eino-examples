package config

import "os"

// ArkChatConfig holds chat model settings.
type ArkChatConfig struct {
	APIKey string
	Base   string
	Region string
	Model  string
}

// LoadArkChatConfig loads chat config from env with defaults.
func LoadArkChatConfig() ArkChatConfig {
	apiKey := os.Getenv("ARK_API_KEY")
	base := os.Getenv("ARK_BASE_URL")
	if base == "" {
		base = "https://ark.cn-beijing.volces.com/api/v3"
	}
	model := os.Getenv("ARK_MODEL")
	if model == "" {
		model = "doubao-seed-1-6-251015"
	}
	region := os.Getenv("ARK_REGION")
	if region == "" {
		region = "cn-beijing"
	}
	return ArkChatConfig{
		APIKey: apiKey,
		Base:   base,
		Region: region,
		Model:  model,
	}
}

// ArkEmbedConfig holds embedding settings.
type ArkEmbedConfig struct {
	APIKey string
	Base   string
	Model  string
}

// LoadArkEmbedConfig loads embedding config from env with defaults.
func LoadArkEmbedConfig() ArkEmbedConfig {
	apiKey := os.Getenv("EMBED_API_KEY")
	base := os.Getenv("EMBED_BASE_URL")
	if base == "" {
		base = "https://ark.cn-beijing.volces.com/api/v3/embeddings/multimodal"
	}
	model := os.Getenv("EMBED_MODEL")
	if model == "" {
		model = "doubao-embedding-vision-250615"
	}
	return ArkEmbedConfig{
		APIKey: apiKey,
		Base:   base,
		Model:  model,
	}
}

