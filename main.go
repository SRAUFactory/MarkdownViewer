package main

import "fmt"
import "github.com/russross/blackfriday"
import "io/ioutil"
import "net/http"

const lenPath = len("/")

func getName(r *http.Request) string {
	param := r.URL.Path[lenPath:]
	return param + ".md"
}

func view(w http.ResponseWriter, r *http.Request) {
	md, err := ioutil.ReadFile(getName(r))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	output := blackfriday.MarkdownCommon([]byte(md))
	fmt.Fprintf(w, string(output))
}

func main() {
	http.HandleFunc("/", view)
	http.ListenAndServe(":8080", nil)
}
