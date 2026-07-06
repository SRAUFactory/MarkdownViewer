package main

import (
	"html/template"
	"path/filepath"
	"strings"
)

// PageData は記事テンプレート（view.html）に渡すデータ構造です。
type PageData struct {
	Title     string
	UpdatedAt string
	Body      template.HTML
}

// parseFrontMatter はMarkdownからYAML Front Matterをパースし、
// メタデータのマップと、メタデータ部分を除去した本文を返します。
func parseFrontMatter(content string) (map[string]string, string) {
	meta := make(map[string]string)
	
	// 改行コードの正規化 (\r\n -> \n)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return meta, content
	}

	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return meta, content
	}

	for i := 1; i < endIdx; i++ {
		line := lines[i]
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `'"`)
			meta[key] = val
		}
	}

	body := strings.Join(lines[endIdx+1:], "\n")
	return meta, body
}

// extractTitle はタイトルを優先順位に従って決定します。
func extractTitle(meta map[string]string, body string, filePath string) string {
	// 優先順位1: フロントマターの title
	if title, ok := meta["title"]; ok && title != "" {
		return title
	}

	// 優先順位2: 本文中の最初の見出し1 (# )
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}

	// 優先順位3: ファイル名（拡張子を除く）
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
