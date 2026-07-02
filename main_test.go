package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFrontMatter(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedMeta map[string]string
		expectedBody string
	}{
		{
			name: "With valid front matter",
			content: "---\ntitle: \"Test Title\"\nupdated_at: '2026-07-02'\n---\n# Header 1\nBody content",
			expectedMeta: map[string]string{
				"title":      "Test Title",
				"updated_at": "2026-07-02",
			},
			expectedBody: "# Header 1\nBody content",
		},
		{
			name: "Without front matter",
			content: "# Header 1\nBody content",
			expectedMeta: map[string]string{},
			expectedBody: "# Header 1\nBody content",
		},
		{
			name: "Incomplete front matter",
			content: "---\ntitle: \"Test Title\"\n# Header 1\nBody content",
			expectedMeta: map[string]string{},
			expectedBody: "---\ntitle: \"Test Title\"\n# Header 1\nBody content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, body := parseFrontMatter(tt.content)

			if len(meta) != len(tt.expectedMeta) {
				t.Errorf("expected meta len %d, got %d", len(tt.expectedMeta), len(meta))
			}
			for k, v := range tt.expectedMeta {
				if meta[k] != v {
					t.Errorf("expected meta[%q] = %q, got %q", k, v, meta[k])
				}
			}
			if body != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, body)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name          string
		meta          map[string]string
		body          string
		filePath      string
		expectedTitle string
	}{
		{
			name:          "Title in meta",
			meta:          map[string]string{"title": "Meta Title"},
			body:          "# Header Title\nBody",
			filePath:      "/path/to/file.md",
			expectedTitle: "Meta Title",
		},
		{
			name:          "Title in header 1",
			meta:          map[string]string{},
			body:          "# Header Title\nBody",
			filePath:      "/path/to/file.md",
			expectedTitle: "Header Title",
		},
		{
			name:          "Title in filename",
			meta:          map[string]string{},
			body:          "Just some body text without header 1\n## Header 2",
			filePath:      "/path/to/my-file-name.md",
			expectedTitle: "my-file-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := extractTitle(tt.meta, tt.body, tt.filePath)
			if title != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, title)
			}
		})
	}
}

func TestView(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "md_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("MARK_DOWN_HOME")
	os.Setenv("MARK_DOWN_HOME", tmpDir)
	defer os.Setenv("MARK_DOWN_HOME", origHome)

	// index.html がカレントディレクトリにある想定のため、テストカレントディレクトリを変更するか、
	// あるいは ../ などのパス解決になりますが、ここではテスト実行ディレクトリがプロジェクトルートである前提です。
	
	err = os.WriteFile(filepath.Join(tmpDir, "meta-present.md"), []byte(`---
title: "Parsed Title"
updated_at: "2026-07-02 12:00:00"
---
# Main Header
Body text.`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "meta-absent-header-present.md"), []byte(`# Only Header
Body text without meta.`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "meta-absent-header-absent.md"), []byte(`Just body text.`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{name}", view)

	tests := []struct {
		name             string
		path             string
		expectedTitle    string
		expectedMeta     string
		expectedBody     string
		shouldNotContain string
	}{
		{
			name:             "With Meta",
			path:             "/meta-present",
			expectedTitle:    "<title>Parsed Title - Markdown Viewer</title>",
			expectedMeta:     "Updated at: 2026-07-02 12:00:00",
			expectedBody:     "Body text.",
			shouldNotContain: "title:",
		},
		{
			name:             "Without Meta, With Header 1",
			path:             "/meta-absent-header-present",
			expectedTitle:    "<title>Only Header - Markdown Viewer</title>",
			expectedMeta:     "Updated at:",
			expectedBody:     "Body text without meta.",
		},
		{
			name:             "Without Meta, Without Header 1",
			path:             "/meta-absent-header-absent",
			expectedTitle:    "<title>meta-absent-header-absent - Markdown Viewer</title>",
			expectedMeta:     "Updated at:",
			expectedBody:     "Just body text.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, bodyStr)
			}

			if !strings.Contains(bodyStr, tt.expectedTitle) {
				t.Errorf("expected title html %q in body, but not found", tt.expectedTitle)
			}
			if tt.expectedMeta != "" && !strings.Contains(bodyStr, tt.expectedMeta) {
				t.Errorf("expected meta text %q in body, but not found", tt.expectedMeta)
			}
			if !strings.Contains(bodyStr, tt.expectedBody) {
				t.Errorf("expected body text %q in body, but not found", tt.expectedBody)
			}
			if tt.shouldNotContain != "" && strings.Contains(bodyStr, tt.shouldNotContain) {
				t.Errorf("found unwanted text %q in body", tt.shouldNotContain)
			}
		})
	}
}

