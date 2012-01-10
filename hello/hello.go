package hello

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"appengine/user"
	"http"
	"template"
	"html"
	"github.com/russross/blackfriday"
)

type Page struct {
	Content string
}

type foo struct {
	Filename, Content string
	Logout            string
}

var uploadTemplate = template.Must(template.ParseFile("upload.html"))
var viewTemplate = template.Must(template.ParseFile("view.html"))

func upload(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Path[len(prefix):]
		c := appengine.NewContext(r)
		if !user.IsAdmin(c) {
			l, err := user.LoginURL(c, "/post/"+filename)
			if err != nil {
				panic(err)
			}
			http.Redirect(w, r, l, 302)
			return
		}
		k := datastore.NewKey(c, "string", filename, 0, nil)
		if r.Method == "GET" {
			s := new(Page)
			datastore.Get(c, k, s)
			l, err := user.LogoutURL(c, "/")
			if err != nil {
				panic(err)
			}
			uploadTemplate.Execute(w, foo{filename, s.Content, l})
			return
		}
		content := r.FormValue("content")
		output := html.EscapeString(content)
		err := memcache.Set(c, &memcache.Item{Key: filename, Value: []byte(output)})
		if err != nil {
			panic(err)
		}
		_, err = datastore.Put(c, k, &Page{output})
		if err != nil {
			panic(err)
		}
		http.Redirect(w, r, "/view/"+filename, 302)
	}
}

func view(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		p := r.URL.Path[len(prefix):]
		s := new(Page)
		k := datastore.NewKey(c, "string", p, 0, nil)
		if item, err := memcache.Get(c, p); err == memcache.ErrCacheMiss {
			datastore.Get(c, k, s)
			err = memcache.Set(c, &memcache.Item{Key: p, Value: []byte(s.Content)})
			if err != nil {
				panic(err)
			}
		} else if err != nil {
			panic(err)
		} else {
			s.Content = string(item.Value)
		}
		output := string(blackfriday.MarkdownCommon([]byte(s.Content)))
		viewTemplate.Execute(w, foo{p, output, ""})
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/view/index", 302)
	} else {
		http.Redirect(w, r, "/view"+r.URL.Path, 302)
	}
}

type route func(string) http.HandlerFunc

func handle(u string, p route) {
	http.Handle(u, p(u))
}

func init() {
	handle("/view/", view)
	handle("/post/", upload)
	http.HandleFunc("/", index)
}
// indent
