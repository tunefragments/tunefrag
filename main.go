package main

import (
	"html/template"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type Post struct {
	Author  string `yaml:"author"`
	Title   string `yaml:"title"`
	Order   int    `yaml:"order"`
	Content template.HTML
}

func LoadFromMarkdownFile(filename string) Post {
	raw, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	poststring := string(raw)
	splited := strings.Split(poststring, "==========")

	var postinfo Post
	err = yaml.Unmarshal([]byte(splited[0]), &postinfo)

	if err != nil {
		panic(err)
	}

	p := parser.NewWithExtensions(parser.CommonExtensions)
	asts := p.Parse([]byte(splited[1]))

	r := html.NewRenderer(html.RendererOptions{Flags: html.CommonFlags | html.HrefTargetBlank})
	postinfo.Content = template.HTML(markdown.Render(asts, r))

	return postinfo
}

func RenderToHTMLTemplates(p Post) {
	t, err := template.ParseFiles("templates/post.html")
	if err != nil {
		panic(err)
	}

	f, err := os.Create("dist/" + p.Title + ".html")
	if err != nil {
		panic(err)
	}

	err = t.Execute(f, p)
	if err != nil {
		panic(err)
	}
}

func LoadAllPosts() []Post {
	files, err := os.ReadDir("posts")
	if err != nil {
		panic(err)
	}

	var posts []Post
	for _, file := range files {
		if !file.IsDir() {
			posts = append(posts, LoadFromMarkdownFile("posts/"+file.Name()))
		}
	}

	return posts
}

func RenderIndex(posts []Post) {
	slices.Reverse(posts)
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		panic(err)
	}

	f, err := os.Create("dist/index.html")
	if err != nil {
		panic(err)
	}

	err = t.Execute(f, posts)
	if err != nil {
		panic(err)
	}
}

func CopyFile(src, dst string) {
	sourceFile, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		panic(err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		panic(err)
	}
}

func main() {
	posts := LoadAllPosts()
	slices.SortFunc(posts, func(a, b Post) int {
		if a.Order < b.Order {
			return -1
		}
		if a.Order > b.Order {
			return 1
		}
		return 0
	})

	for _, post := range posts {
		RenderToHTMLTemplates(post)
	}
	RenderIndex(posts)
	CopyFile("templates/style.css", "dist/style.css")
}
