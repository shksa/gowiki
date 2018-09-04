package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

// 18-1008815285
const packageDir = "~/go/src/github.com/shksa/learninggowiki"

// Page is a custom structure type that stores title and the body of a wiki.
type Page struct {
	Title string
	Body  []byte
}

// ViewTemplatePage is a custom structure type that stores Title and the HTML body specifially for the view template page
type ViewTemplatePage struct {
	Title string
	Body  template.HTML
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile(filepath.Join(packageDir, "data", filename), p.Body, 0600)
}

func load(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(filepath.Join(packageDir, "data", filename))
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

/* Title validation
1. The function regexp.MustCompile will parse and compile the regular expression, and return a regexp.Regexp.
2. MustCompile is distinct from Compile in that it will panic if the expression compilation fails,
while Compile returns an error as a second parameter.
*/
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

/* Template caching
1. renderTemplate should not call ParseFiles every time when a page needs to be rendered.
2. A better approach would be to call ParseFiles once at program initialization,
parsing all templates into a single *Template.
3. Then we can use the ExecuteTemplate method to render a specific template.
4. First we create a global variable named templates, and initialize it with ParseFiles.
5. The function template.Must is a convenience wrapper that panics when passed a non-nil error value,
and otherwise returns the *Template unaltered. A panic is appropriate here; if the templates can't be
loaded the only sensible thing to do is exit the program.
6. The ParseFiles function takes any number of string arguments that identify our template files,
and parses those files into templates that are named after the base file name.
6. So the template name is the template file name.
*/

var templates = template.Must(template.ParseFiles(
	filepath.Join(packageDir, "tmpl", "edit.html"),
	filepath.Join(packageDir, "tmpl", "view.html"),
	filepath.Join(packageDir, "tmpl", "frontPage.html"),
))

func renderTemplate(w http.ResponseWriter, templateFilename string, data interface{}) {
	err := templates.ExecuteTemplate(w, templateFilename, data)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderViewTemplate(w http.ResponseWriter, templateFilename string, pageData *Page) {
	viewTemplatePageData := ViewTemplatePage{Title: pageData.Title}

	viewTemplatePageData.Body = template.HTML(
		availableTitlesRegExp.ReplaceAllStringFunc(
			string(pageData.Body),
			func(match string) string {
				replacementOfMatch := fmt.Sprintf(`<a href="/view/%s">%s</a>`, match, match)
				return replacementOfMatch
			},
		),
	)

	renderTemplate(w, templateFilename, viewTemplatePageData)
}

/*  Using decorators to reduce code duplication.
1. Validating and catching the error condition for title in each handler introduces a lot of repeated code.
2. What if we could wrap each of the handlers in a function that does this validation and error checking?
3. Go's function literals provide a powerful means of abstracting functionality that can help us here.
4. The closure returned by makeHandler is a function that takes an http.ResponseWriter and http.Request
(in other words, an http.HandlerFunc).
5. The closure extracts the title from the request path, and validates it with the validPath regexp.
6. If the title is invalid, an error will be written to the ResponseWriter using the http.NotFound function.
7. If the title is valid, the enclosed handler function fn will be called with the ResponseWriter, Request, and
title as arguments.
*/

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		match := validPath.FindStringSubmatch(r.URL.Path)
		if match == nil {
			http.NotFound(w, r)
			return
		}
		title := match[2]
		fn(w, r, title)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	pageData, err := load(title)
	if err != nil {
		// http.Redirect replies to the request with a redirect to url.
		// The http.Redirect function adds an HTTP status code of http.StatusFound (302)
		// and a Location header ("/edit/theTitle") to the HTTP response.
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderViewTemplate(w, "view.html", pageData)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	pageData, err := load(title)
	if err != nil {
		pageData = &Page{Title: title}
	}
	renderTemplate(w, "edit.html", pageData)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	// <textarea name="body" rows="20" cols="80">
	body := r.FormValue("body")
	newPageData := &Page{Title: title, Body: []byte(body)}
	// save() writes the new page data to file
	err := newPageData.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// update the wiki title list if the current title isn't already present
	if isAlreadyPresent := availableWikiTitles[title]; !isAlreadyPresent {
		updateWikiTitleList(title)
		updateWikiTitlesRexEx(title)
	}
	// client is redirected to the /view/ page.
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "frontPage.html", availableWikiTitles)
}

// for page inter-linking
var availableTitlesPattern = ""
var availableTitlesRegExp *regexp.Regexp

// for inter-linking new page titles
func updateWikiTitlesRexEx(title string) {
	availableTitlesPattern += fmt.Sprintf("|%s", title)
	availableTitlesRegExp = regexp.MustCompile("(" + availableTitlesPattern + ")")
}

// for front page listing
var availableWikiTitles = make(map[string]bool)

func updateWikiTitleList(title string) {
	availableWikiTitles[title] = true
}

func init() {
	files, err := ioutil.ReadDir(filepath.Join(packageDir, "data"))
	if err != nil {
		log.Fatal("could not read files from the ~/go/src/github.com/shksa/gowiki/data directory due to error:\n" + err.Error())
	}
	for _, file := range files {
		title := strings.Split(file.Name(), ".")[0] // bcoz ".txt" should not be included in the title
		availableTitlesPattern += fmt.Sprintf("%s|", title)
		availableWikiTitles[title] = true
	}
	availableTitlesPattern = availableTitlesPattern[:len(availableTitlesPattern)-1]
	availableTitlesRegExp = regexp.MustCompile("(" + availableTitlesPattern + ")")
	// fmt.Println(availableTitlesPattern)
	// fmt.Printf("%s\n", availableTitlesRegExp.ReplaceAllFunc([]byte("messi president of america is donaldTrump. He is pretty test."), func(match []byte) []byte {
	// 	replacementOfMatch := fmt.Sprintf(`<a href="/view/%s">%s</a>`, match, match)
	// 	return []byte(replacementOfMatch)
	// }))
}

/*
1. A wiki has the ability to view and edit pages.
2. If the requested Page doesn't exist, it should redirect the client to the edit Page so the content may be created.
*/
func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
