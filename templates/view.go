package main

import (
	"bytes"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type TemplateData struct {
	CurrentView    string
	LoggedIn       bool
	CanUpdatePatch bool
	UserName       string
	StaticRoot     string
	Headline       string
	HeadlinePrefix string
	FinalJS        template.JS
	Patch          MasherPatch
	Patches        []MasherPatch
	Referer        string
	ViewingUser    MasherUser
	Tutorial       template.HTML
}

var masherTemplates *template.Template
var tutorialTemplates *template.Template

func InitTemplates() {
	var err error
	funcs := template.FuncMap{
		"HumanDate": func(d int64) string {
			return time.Unix(d, 0).Format("2006/01/02")
		},
	}
	masherTemplates, err = template.New("masher").Funcs(funcs).ParseFiles(
		"templates/layout.html",
		"templates/patch.html",
		"templates/user.html",
		"templates/browse.html",
		"templates/about.html",
		"templates/learn.html",
		"templates/404.html")
	if err != nil {
		panic(err)
	}
	tutorialTemplates, err = template.New("masher").Funcs(funcs).ParseGlob("templates/tutorial/*.html")
	if err != nil {
		panic(err)
	}
}

func BaseTemplateData(w http.ResponseWriter, r *http.Request) TemplateData {
	ret := TemplateData{
		StaticRoot: MasherConfig.StaticRoot}
	session, err := SessionStore.Get(r, "sess")
	if err != nil {
		panic(err)
	}
	if ret.UserName, ret.LoggedIn = session.Values["name"].(string); !ret.LoggedIn {
		ret.UserName = ""
	}
	return ret
}

func ViewPatch(w http.ResponseWriter, r *http.Request) {
	var err error
	params := mux.Vars(r)
	data := BaseTemplateData(w, r)
	data.Patch, err = RetrievePatch(params["id"])
	if err != nil {
		ViewNotFound(w, r)
		return
	}
	if data.Patch.Author == data.UserName {
		data.CanUpdatePatch = true
	}
	data.CurrentView = "Patch"
	data.Headline = data.Patch.Title + " by " + data.Patch.Author
	err = masherTemplates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		panic(err)
	}
}

func ViewUser(w http.ResponseWriter, r *http.Request) {
	var err error
	params := mux.Vars(r)
	data := BaseTemplateData(w, r)
	data.ViewingUser, err = RetrieveUser(params["user"])
	if err != nil {
		ViewNotFound(w, r)
		return
	}
	data.Patches, err = RetrievePatches(SearchFilter{Author: data.ViewingUser.Name})
	if err != nil {
		ViewNotFound(w, r)
		return
	}
	sort.Slice(data.Patches, func(i, j int) bool {
		return data.Patches[i].DateCreated > data.Patches[j].DateCreated
	})
	data.HeadlinePrefix = "Viewing: "
	data.Headline = "patches by " + data.ViewingUser.Name
	data.CurrentView = "User"
	err = masherTemplates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		panic(err)
	}
}

func ViewContinue(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(w, r)
	data.CurrentView = "Patch"
	data.HeadlinePrefix = "Editing: "
	data.Headline = "new patch"
	data.Patch.Files = map[string]string{"main.sp": "# You have just one Sporth. Make something.\n\n"}
	data.FinalJS = "  restoreAutosave(); "
	err := masherTemplates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		panic(err)
	}
}

func ViewNew(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(w, r)
	data.CurrentView = "Patch"
	data.HeadlinePrefix = "Editing: "
	data.Headline = "new patch"
	data.Patch.Files = map[string]string{"main.sp": "# You have just one Sporth. Make something.\n\n"}
	err := masherTemplates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		panic(err)
	}
}

func ViewBrowse(w http.ResponseWriter, r *http.Request) {
	var err error
	data := BaseTemplateData(w, r)
	data.Patches, err = RetrievePatches(SearchFilter{})
	if err != nil {
		ViewNotFound(w, r)
		return
	}
	sort.Slice(data.Patches, func(i, j int) bool {
		return data.Patches[i].DateCreated > data.Patches[j].DateCreated
	})
	data.HeadlinePrefix = "Viewing: "
	data.Headline = "all patches"
	data.CurrentView = "Browse"
	err = masherTemplates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		panic(err)
	}
}

func ViewAbout(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(w, r)
	data.Headline = "About"
	data.CurrentView = "About"
	err := masherTemplates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		panic(err)
	}
}

func ViewLearn(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tutPage := params["page"]
	if tutPage == "" {
		tutPage = "index"
	}
	tutPage = tutPage + ".html"
	tutorialBuf := &bytes.Buffer{}
	data := BaseTemplateData(w, r)

	data.CurrentView = "Learn"
	err := tutorialTemplates.ExecuteTemplate(tutorialBuf, tutPage, data)
	if err != nil {
		ViewNotFound(w, r)
		return
	}
	var title = strings.Split(strings.SplitAfter(tutorialBuf.String(), "<title>")[1], "</title>")[0]
	if title == "" {
		title = strings.Split(strings.SplitAfter(tutorialBuf.String(), "<h1>")[1], "</h1>")[0]
	}
	data.Headline = title
	data.Tutorial = template.HTML(strings.Split(strings.SplitAfter(tutorialBuf.String(), "<body>")[1], "</body>")[0])
	data.Tutorial = "<!-- start content -->" + data.Tutorial + "<!-- end content -->"

	err = masherTemplates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		panic(err)
	}
}

func ViewNotFound(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(w, r)
	data.Headline = "404 - Not Found"
	data.CurrentView = "404"
	w.WriteHeader(http.StatusNotFound)
	err := masherTemplates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		panic(err)
	}
}
