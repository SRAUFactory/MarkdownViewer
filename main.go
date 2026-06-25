package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/russross/blackfriday"
)

func view(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name") // Go 1.22+ パスパラメータ
	if name == "" {
		name = "index"
	}

	var htmlContent template.HTML
	if name != "index" {
		home := os.Getenv("MARK_DOWN_HOME")
		if home == "" {
			home = "."
		}
		filePath := filepath.Join(home, name+".md")

		md, err := os.ReadFile(filePath)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		output := blackfriday.MarkdownCommon(md)
		htmlContent = template.HTML(string(output))
	}

	t, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	t.Execute(w, htmlContent)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", view)
	mux.HandleFunc("GET /{name}", view)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
