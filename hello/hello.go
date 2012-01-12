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

type Foo struct {
	Filename, Content string
	Logout            string
}

var uploadTemplate = template.Must(template.ParseFile("upload.html"))
var viewTemplate = template.Must(template.ParseFile("view.html"))
var deleteTemplate = template.Must(template.ParseFile("delete.html"))

func upload(prefix string) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Path[len(prefix):]
		c := appengine.NewContext(r)
		if !user.IsAdmin(c) {
			l, err := user.LoginURL(c, prefix+filename)
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
			l, err := user.LogoutURL(c, "/view/"+filename)
			if err != nil {
				panic(err)
			}
			uploadTemplate.Execute(w, Foo{filename, s.Content, l})
			return
		}
		if r.Method != "POST" {
			panic("Invalid method")
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
	return http.HandlerFunc(f)
}

func view(prefix string) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		p := r.URL.Path[len(prefix):]
		if p == "" {
			p = "index"
		}
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
		viewTemplate.Execute(w, Foo{p, output, ""})
	}
	return http.HandlerFunc(f)
}

func delete(prefix string) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Path[len(prefix):]
		c := appengine.NewContext(r)
		if !user.IsAdmin(c) {
			l, err := user.LoginURL(c, prefix+filename)
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
			l, err := user.LogoutURL(c, "/view/"+filename)
			if err != nil {
				panic(err)
			}
			deleteTemplate.Execute(w, Foo{filename, s.Content, l})
			return
		}
		if r.Method != "POST" {
			panic("Invalid method")
		}
		err := memcache.Delete(c, filename)
		if err != nil {
			panic(err)
		}
		err = datastore.Delete(c, k)
		if err != nil {
			panic(err)
		}
		http.Redirect(w, r, "/view/"+filename, 302)
	}
	return http.HandlerFunc(f)
}

type route func(string) http.Handler

func handle(u string, p route) {
	http.Handle(u, p(u))
}

func init() {
	handle("/view/", view)
	handle("/post/", upload)
	handle("/delete/", delete)
	handle("/", view)
}
