package main

import "fmt"
import "github.com/russross/blackfriday"
import "io/ioutil"
import "net/http"

func view(w http.ResponseWriter, req *http.Request) {
	md, err := ioutil.ReadFile("test.md")
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
