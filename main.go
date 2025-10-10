package main

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const postsPerPage = 10

type Post struct {
	Author  string `yaml:"author"`
	Title   string `yaml:"title"`
	Order   int    `yaml:"order"`
	Content template.HTML
}

type Page struct {
	Posts       []Post
	CurrentPage int
	TotalPages  int
	HasNext     bool
	HasPrev     bool
	NextUrl     string
	PrevUrl     string
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

func RenderIndex(page Page) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		panic(err)
	}

	var filename string
	if page.CurrentPage == 1 {
		filename = "dist/index.html"
	} else {
		filename = fmt.Sprintf("dist/page/%d.html", page.CurrentPage)
	}

	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	err = t.Execute(f, page)
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

func CopyStaticFiles(sourceDir, destDir string) {
	os.MkdirAll(destDir, os.ModePerm)

	files, err := os.ReadDir(sourceDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if !file.IsDir() && !strings.HasSuffix(file.Name(), ".html") {
			sourceFile := filepath.Join(sourceDir, file.Name())
			destFile := filepath.Join(destDir, file.Name())
			CopyFile(sourceFile, destFile)
		}
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
	slices.Reverse(posts)

	for _, post := range posts {
		RenderToHTMLTemplates(post)
	}

	os.MkdirAll("dist/page", os.ModePerm)
	totalPages := int(math.Ceil(float64(len(posts)) / float64(postsPerPage)))

	for i := 1; i <= totalPages; i++ {
		start := (i - 1) * postsPerPage
		end := start + postsPerPage
		if end > len(posts) {
			end = len(posts)
		}

		pagePosts := posts[start:end]

		prevUrl := ""
		if i > 1 {
			if i == 2 {
				prevUrl = "../index.html"
			} else {
				prevUrl = fmt.Sprintf("page/%d.html", i-1)
			}
		}

		nextUrl := ""
		if i < totalPages {
			nextUrl = fmt.Sprintf("page/%d.html", i+1)
		}

		page := Page{
			Posts:       pagePosts,
			CurrentPage: i,
			TotalPages:  totalPages,
			HasPrev:     i > 1,
			HasNext:     i < totalPages,
			PrevUrl:     prevUrl,
			NextUrl:     nextUrl,
		}
		RenderIndex(page)
	}

	CopyStaticFiles("templates", "dist")
}
