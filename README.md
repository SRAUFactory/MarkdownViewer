# MarkdownViewer

This project is a tool for displaying Markdown format files as HTML on web browser.

## INSTALL && RUN
```
cd $GOPAH/src
go get github.com/russross/blackfriday
go get github.com/SRAUFactory/MarkdownViewer
go run github.com/SRAUFactory/MarkdownViewer/main.go
```

## Setting environment variables
Set Markdown file placement directory to environment variable.
```
export $MARK_DOWN_HOME="~/Markdown"
```

## Display
Enter the following URL in the web browser and display it.
```
format)
http://localhost:8080/[Markdown File Prefix]

ex)
http://localhost:8080/init
```

If the specified file doesn't exist in `$ MARK_DOWN_HOME`,` README.md` is displayed.
