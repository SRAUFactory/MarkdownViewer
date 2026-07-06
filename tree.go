package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

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

// buildFileTree は指定されたルートディレクトリ以下のMarkdownファイルをスキャンしてツリー構造を作成します。
func buildFileTree(rootDir string) (*FileNode, error) {
	root := &FileNode{
		Name:  "Root",
		IsDir: true,
	}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == rootDir {
			return nil
		}

		// 隠しフォルダ・ファイルをスキップ
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		parts := strings.Split(rel, string(filepath.Separator))

		if d.IsDir() {
			insertNode(root, parts, "", true)
		} else if strings.HasSuffix(d.Name(), ".md") {
			urlPath := strings.TrimSuffix(rel, ".md")
			// Windows環境のパスセパレータ（\）をスラッシュ（/）に統一
			urlPath = strings.ReplaceAll(urlPath, string(filepath.Separator), "/")
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
