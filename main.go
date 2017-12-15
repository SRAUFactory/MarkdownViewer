package main

import "fmt"
import "github.com/russross/blackfriday"
import "io/ioutil"

func main() {
	md, err := ioutil.ReadFile("test.md")
	if err != nil {
		panic(err)
	}
	output := blackfriday.MarkdownCommon([]byte(md))
	fmt.Println(string(output))
}
