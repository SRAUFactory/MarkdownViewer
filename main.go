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

// PageData は記事テンプレート（view.html）に渡すデータ構造です。
type PageData struct {
	Title     string
	UpdatedAt string
	Body      template.HTML
}

// FileNode はファイルシステムのツリー構造を表現します。
type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*FileNode
}

// IndexData はインデックスページ（index.html）に渡すデータ構造です。
type IndexData struct {
	Tree *FileNode
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

// buildFileTree は指定されたルートディレクトリ以下のMarkdownファイルをスキャンしてツリー構造を作成します。
func buildFileTree(rootDir string) (*FileNode, error) {
	root := &FileNode{
		Name:  "Root",
		IsDir: true,
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == rootDir {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		parts := strings.Split(rel, string(filepath.Separator))
		// 隠しフォルダ・ファイルをスキップ
		for _, part := range parts {
			if strings.HasPrefix(part, ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			insertNode(root, parts, "", true)
		} else if strings.HasSuffix(info.Name(), ".md") {
			urlPath := strings.TrimSuffix(rel, ".md")
			insertNode(root, parts, urlPath, false)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Markdownファイルを一つも含まない空のディレクトリをトリミング
	pruneEmptyDirs(root)

	return root, nil
}

// pruneEmptyDirs は子孫ノードにファイル（.md）が1つもないディレクトリノードをツリーから削除します。
// 有効なファイルまたは有効な子要素を持つ場合は true を返します。
func pruneEmptyDirs(node *FileNode) bool {
	if !node.IsDir {
		return true
	}

	var activeChildren []*FileNode
	for _, child := range node.Children {
		if pruneEmptyDirs(child) {
			activeChildren = append(activeChildren, child)
		}
	}
	node.Children = activeChildren

	return len(node.Children) > 0
}

func insertNode(root *FileNode, parts []string, urlPath string, isDir bool) {
	current := root
	for i, part := range parts {
		isLast := i == len(parts)-1
		
		targetName := part
		if !isDir && isLast && strings.HasSuffix(part, ".md") {
			targetName = strings.TrimSuffix(part, ".md")
		}

		var found *FileNode
		for _, child := range current.Children {
			if child.Name == targetName && child.IsDir == (!isLast || isDir) {
				found = child
				break
			}
		}

		if found == nil {
			node := &FileNode{
				Name:  targetName,
				IsDir: !isLast || isDir,
			}
			if !node.IsDir {
				node.Path = urlPath
			}
			current.Children = append(current.Children, node)
			current = node
		} else {
			current = found
		}
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	home := os.Getenv("MARK_DOWN_HOME")
	if home == "" {
		home = "."
	}

	tree, err := buildFileTree(home)
	if err != nil {
		http.Error(w, "Failed to build file tree", http.StatusInternalServerError)
		return
	}

	t, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	t.Execute(w, IndexData{Tree: tree})
}

func view(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

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

	t, err := template.ParseFiles("view.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", index)
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

