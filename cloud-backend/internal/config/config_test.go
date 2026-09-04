package config

import "testing"

func TestEmbeddingReadyLocalNeedsNoKey(t *testing.T) {
	ok := Config{EmbeddingBaseURL: "http://127.0.0.1:11434"}
	if !ok.EmbeddingReady() {
		t.Fatal("local ollama should be ready without a key")
	}
	remote := Config{EmbeddingBaseURL: "https://api.openai.com"}
	if remote.EmbeddingReady() {
		t.Fatal("non-local embeddings still need a key")
	}
	remote.EmbeddingAPIKey = "k"
	if !remote.EmbeddingReady() {
		t.Fatal("hosted with key")
	}
	empty := Config{}
	if empty.EmbeddingReady() {
		t.Fatal("missing base URL")
	}
}
