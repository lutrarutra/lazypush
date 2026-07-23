package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lutrarutra/lazypush/internal/llm"
)

func TestGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s, want /chat/completions", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("auth = %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"Hello, world!"}}]}`))
	}))
	defer server.Close()

	client := llm.New(server.URL, "sk-test", "gpt-4")
	result, err := client.Generate(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result != "Hello, world!" {
		t.Errorf("Generate() = %q, want %q", result, "Hello, world!")
	}
}

func TestGenerateAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer server.Close()

	client := llm.New(server.URL, "bad-key", "gpt-4")
	_, err := client.Generate(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("Generate() expected error, got nil")
	}
}

func TestGenerateEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := llm.New(server.URL, "sk-test", "gpt-4")
	_, err := client.Generate(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("Generate() expected error for empty choices, got nil")
	}
}

func TestCommitMessagePrompt(t *testing.T) {
	diff := "--- a/foo.go\n+++ b/foo.go\n@@ -1 +1 @@\n-func old()\n+func new()"
	sys, user := llm.CommitMessagePrompt(diff)
	if sys == "" {
		t.Error("system prompt is empty")
	}
	if user == "" {
		t.Error("user prompt is empty")
	}
}

func TestPRDescriptionPrompt(t *testing.T) {
	diff := "--- a/foo.go\n+++ b/foo.go\n@@ -1 +1 @@\n-func old()\n+func new()"
	sys, user := llm.PRDescriptionPrompt(diff)
	if sys == "" {
		t.Error("system prompt is empty")
	}
	if user == "" {
		t.Error("user prompt is empty")
	}
}
