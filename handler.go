package main

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday"
)

// AppHandler はアプリケーション全体の状態（テンプレート、設定など）を保持します。
type AppHandler struct {
	IndexTemplate *template.Template
	ViewTemplate  *template.Template
	MarkdownHome  string
}

// NewAppHandler は新しい AppHandler を初期化して返します。
func NewAppHandler(homeDir string) (*AppHandler, error) {
	indexTmpl, err := template.ParseFiles("index.html")
	if err != nil {
		return nil, err
	}

	viewTmpl, err := template.ParseFiles("view.html")
	if err != nil {
		return nil, err
	}

	return &AppHandler{
		IndexTemplate: indexTmpl,
		ViewTemplate:  viewTmpl,
		MarkdownHome:  homeDir,
	}, nil
}

func (h *AppHandler) Index(w http.ResponseWriter, r *http.Request) {
	tree, err := buildFileTree(h.MarkdownHome)
	if err != nil {
		http.Error(w, "Failed to build file tree", http.StatusInternalServerError)
		return
	}

	h.IndexTemplate.Execute(w, IndexData{Tree: tree})
}

func (h *AppHandler) View(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	absHome, err := filepath.Abs(h.MarkdownHome)
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
	var data PageData
	data.Title = extractTitle(meta, body, absFilePath)

	// updated_at の設定
	if updatedAt, ok := meta["updated_at"]; ok && updatedAt != "" {
		data.UpdatedAt = updatedAt
	} else {
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

	h.ViewTemplate.Execute(w, data)
}
