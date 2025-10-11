package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/gosimple/slug"
)

const postsPerPage = 4

type Post struct {
	Author  string `yaml:"author"`
	Title   string `yaml:"title"`
	Order   int    `yaml:"order"`
	Content template.HTML
	Slug    string
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

	postinfo.Slug = slug.Make(postinfo.Title)

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

	f, err := os.Create("dist/" + p.Slug + ".html")
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
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
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

	filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			os.MkdirAll(destPath, os.ModePerm)
		} else {
			if !strings.HasSuffix(path, ".html") {
				CopyFile(path, destPath)
			}
		}

		return nil
	})
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
	CopyStaticFiles("posts/statics", "dist/statics")
}
