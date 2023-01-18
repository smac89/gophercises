package cyoa

import (
	"gophercises.com/cyoa"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

type HandlerOption struct {
	tpl    *template.Template
	pathFn func(r *http.Request) string
}

type storyHandler struct {
	story cyoa.Story
	HandlerOption
}

var storyTemplate *template.Template

func defaultPathFn(r *http.Request) string {
	chapterPath := strings.TrimSpace(r.URL.Path)
	if chapterPath == "" || chapterPath == "/" {
		chapterPath = "/intro"
	}
	return chapterPath[1:]
}

func init() {
	fp := path.Join("web/templates", "story.gohtml")
	storyTemplate = template.Must(template.ParseFiles(fp))
}

func NewStoryHandler(s cyoa.Story, opt *HandlerOption) http.Handler {
	handler := storyHandler{story: s, HandlerOption: HandlerOption{tpl: storyTemplate, pathFn: defaultPathFn}}
	if opt != nil {
		if opt.tpl != nil {
			handler.tpl = opt.tpl
		}
		if opt.pathFn != nil {
			handler.pathFn = opt.pathFn
		}
	}
	return &handler
}

func (storyHandler *storyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	chapterPath := storyHandler.pathFn(r)

	if chapter, ok := storyHandler.story[chapterPath]; ok {
		err := storyHandler.tpl.Execute(w, chapter)
		if err != nil {
			log.Printf("error rendering template for story '%s'", chapterPath)
			http.Error(w, "Something went wrong...", http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "Chapter not found", http.StatusNotFound)
	}
}
