package config

import "os"

// QdrantConfig holds configuration for connecting to a Qdrant instance.
type QdrantConfig struct {
	URL        string
	Collection string
}

// LoadQdrantConfig reads Qdrant configuration from environment variables with sane defaults.
//
//   - QDRANT_URL:        base URL of Qdrant HTTP endpoint (default: http://localhost:6333)
//   - QDRANT_COLLECTION: collection name for storing/searching novels (default: novels)
func LoadQdrantConfig() QdrantConfig {
	url := os.Getenv("QDRANT_URL")
	if url == "" {
		url = "http://localhost:6333"
	}

	coll := os.Getenv("QDRANT_COLLECTION")
	if coll == "" {
		coll = "novels"
	}

	return QdrantConfig{
		URL:        url,
		Collection: coll,
	}
}


