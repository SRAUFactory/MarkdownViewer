package main

import "fmt"
import "github.com/russross/blackfriday"
import "io/ioutil"
import "net/http"

const lenPath = len("/")

func view(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[lenPath:]
	md, err := ioutil.ReadFile(name + ".md")
	if err != nil {
		panic(err)
	}
	output := blackfriday.MarkdownCommon([]byte(md))
	fmt.Fprintf(w, string(output))
}

func main() {
	http.HandleFunc("/", view)
	http.ListenAndServe(":8080", nil)
}
