package main

import "github.com/russross/blackfriday"
import "html/template"
import "io/ioutil"
import "net/http"
import "os"

const lenPath = len("/")

func getName(r *http.Request) string {
	param := r.URL.Path[lenPath:]
	home := os.Getenv("MARK_DOWN_HOME")
	return home + "/" + param + ".md"
}

func view(w http.ResponseWriter, r *http.Request) {
	md, err := ioutil.ReadFile(getName(r))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
	}
	output := blackfriday.MarkdownCommon([]byte(md))
	t := template.Must(template.ParseFiles("index.html"))
	t.Execute(w, template.HTML(string(output)))
}

func main() {
	http.HandleFunc("/", view)
	http.ListenAndServe(":8080", nil)
}
