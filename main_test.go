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

func TestIndex(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "md_test_index")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("MARK_DOWN_HOME")
	os.Setenv("MARK_DOWN_HOME", tmpDir)
	defer os.Setenv("MARK_DOWN_HOME", origHome)

	// テスト用ファイル作成
	err = os.WriteFile(filepath.Join(tmpDir, "file1.md"), []byte("# File 1"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Mkdir(filepath.Join(tmpDir, "folder1"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "folder1", "file2.md"), []byte("# File 2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 空のフォルダ（トリミング対象）
	err = os.Mkdir(filepath.Join(tmpDir, "empty_folder"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// 非Markdownファイルのみを含むフォルダ（トリミング対象）
	err = os.Mkdir(filepath.Join(tmpDir, "non_md_folder"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "non_md_folder", "readme.txt"), []byte("not markdown"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	index(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, bodyStr)
	}

	// ツリー形式のファイルリストがHTMLに含まれていることを検証
	if !strings.Contains(bodyStr, "folder1") {
		t.Errorf("expected 'folder1' in index page body, but not found")
	}
	if !strings.Contains(bodyStr, "file1") {
		t.Errorf("expected 'file1' in index page body, but not found")
	}
	if !strings.Contains(bodyStr, "folder1/file2") {
		t.Errorf("expected 'folder1/file2' in index page body, but not found")
	}

	// 空のフォルダや非Markdownフォルダが表示されていないことを検証
	if strings.Contains(bodyStr, "empty_folder") {
		t.Errorf("did not expect 'empty_folder' in index page body, but found")
	}
	if strings.Contains(bodyStr, "non_md_folder") {
		t.Errorf("did not expect 'non_md_folder' in index page body, but found")
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

	// サブディレクトリのテスト用ファイル
	err = os.Mkdir(filepath.Join(tmpDir, "sub"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "sub", "sub-file.md"), []byte(`---
title: "Sub File"
---
Sub body.`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{name...}", view)

	tests := []struct {
		name             string
		path             string
		expectedTitle    string
		expectedMeta     string
		expectedBody     string
		shouldNotContain string
		expectedStatus   int
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
		{
			name:             "With Subdirectory",
			path:             "/sub/sub-file",
			expectedTitle:    "<title>Sub File - Markdown Viewer</title>",
			expectedMeta:     "",
			expectedBody:     "Sub body.",
		},
		{
			name:             "Traversal Attack",
			path:             "/../escaped-file",
			expectedStatus:   http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			if tt.name == "Traversal Attack" {
				req.SetPathValue("name", "../escaped-file")
				view(w, req)
			} else {
				mux.ServeHTTP(w, req)
			}

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			expectedStatus := tt.expectedStatus
			if expectedStatus == 0 {
				expectedStatus = http.StatusOK
			}

			if resp.StatusCode != expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", expectedStatus, resp.StatusCode, bodyStr)
			}

			if expectedStatus == http.StatusOK {
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
			}
		})
	}
}

