package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday"
)

// PageData はテンプレートに渡すデータ構造です。
type PageData struct {
	Title     string
	UpdatedAt string
	Body      template.HTML
	IsIndex   bool
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

func view(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name") // Go 1.22+ パスパラメータ
	if name == "" {
		name = "index"
	}

	var data PageData
	data.Title = "Markdown Viewer"
	data.IsIndex = (name == "index")

	if name != "index" {
		home := os.Getenv("MARK_DOWN_HOME")
		if home == "" {
			home = "."
		}
		
		absHome, err := filepath.Abs(home)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		filePath := filepath.Join(absHome, name+".md")
		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// ディレクトリトラバーサル防止チェック
		if !strings.HasPrefix(absFilePath, absHome) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		fileInfo, err := os.Stat(absFilePath)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		mdBytes, err := os.ReadFile(absFilePath)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		meta, body := parseFrontMatter(string(mdBytes))
		data.Title = extractTitle(meta, body, absFilePath)

		// updated_at の設定
		if updatedAt, ok := meta["updated_at"]; ok && updatedAt != "" {
			data.UpdatedAt = updatedAt
		} else {
			// 優先順位: 最終更新日時 => 作成日時
			modTime := fileInfo.ModTime()
			birthTime := getBirthTime(fileInfo)
			
			if !modTime.IsZero() {
				data.UpdatedAt = modTime.Format("2006-01-02 15:04:05")
			} else if !birthTime.IsZero() {
				data.UpdatedAt = birthTime.Format("2006-01-02 15:04:05")
			}
		}

		output := blackfriday.MarkdownCommon([]byte(body))
		data.Body = template.HTML(string(output))
	}

	t, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{name...}", view)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

