package config

// Config collects all runtime configuration for the demo.
// If需要调整默认值，集中改这里，避免散落在各处。
type Config struct {
	ArkChat  ArkChatConfig
	ArkEmbed ArkEmbedConfig
	Qdrant   QdrantConfig
	Feedback FeedbackConfig
}

// FeedbackConfig controls feedback persistence.
type FeedbackConfig struct {
	LogPath string
}

// LoadConfig aggregates sub-configs. 当前默认依然使用原有 env 默认值的加载器。
func LoadConfig() Config {
	return Config{
		ArkChat:  LoadArkChatConfig(),
		ArkEmbed: LoadArkEmbedConfig(),
		Qdrant:   LoadQdrantConfig(),
		Feedback: FeedbackConfig{
			LogPath: "/tmp/novel_feedback.log",
		},
	}
}
