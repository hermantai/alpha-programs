package mycache

import (
	"appengine"
	"appengine/memcache"

	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

const initialPageHtml = `
<!DOCTYPE html>
<html>
<head>
	<title>My Cache</title>
</head>
<body>
<h1>Add</h1>
<form action="/add" method="post">
Key: <input type="text" name="cache-key" /><br />
Value: <input type="text" name="cache-value" /><br />
<button type="submit">Add</button>
</form>

<h1>Get</h1>
<form action="/get" method="get">
Key: <input type="text" name="cache-key" /><br />
<button type="submit">Get</button>
</form>

<h1>Delete</h1>
<form action="/delete" method="get">
Key: <input type="text" name="cache-key" /><br />
<button type="submit">Delete</button>
</form>

<h1>List</h1>
<form action="/list" method="get">
<button type="submit">List</button>
</form>

<h1>Request</h1>
<p>{{.RemoteAddr}}</p>
<h2>Headers</h2>
{{range $key, $value := .Header}}
<p>{{$key}} => {{$value}}</p>
{{end}}
</body>
</html>
`

const addPageHtml = `
<!DOCTYPE html>
<html>
<head>
	<title>My Cache - Add</title>
</head>
<body>
<p>
Added {{.CacheKey}} => {{.CacheValue}}
</p>
<a href="/">Go back home</a>
</body>
</html>
`

const getPageHtml = `
<!DOCTYPE html>
<html>
<head>
	<title>My Cache - Get</title>
</head>
<body>
<p>{{.CacheKey}} => {{.CacheValue}}</p>
<a href="/">Go back home</a>
</body>
</html>
`

const deletePageHtml = `
<!DOCTYPE html>
<html>
<head>
	<title>My Cache - Delete</title>
</head>
<body>
<p>Key {{.CacheKey}} is deleted.</p>
<a href="/">Go back home</a>
</body>
</html>
`

const listPageHtml = `
<!DOCTYPE html>
<html>
<head>
	<title>My Cache - List</title>
</head>
<body>
{{range $key, $value := .Data}}
<p>{{$key}} => {{$value}}</p>
{{end}}
<a href="/">Go back home</a>
</body>
</html>
`

// System keys
const (
	keyAllKeys = "MYCACHE_ALL_KEYS"
)

type homePage struct {
	RemoteAddr string
	Header     http.Header
}

type addPage struct {
	CacheKey   string
	CacheValue string
}

type getPage struct {
	CacheKey   string
	CacheValue string
}

type deletePage struct {
	CacheKey string
}

type listPage struct {
	Data map[string]string
}

func init() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/list", listHandler)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	hp := homePage{
		r.RemoteAddr,
		r.Header,
	}
	t, err := template.New("home-page").Parse(initialPageHtml)
	check(err)

	t.Execute(w, hp)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	ap := addPage{
		CacheKey:   r.FormValue("cache-key"),
		CacheValue: r.FormValue("cache-value"),
	}

	var allKeys []string

	c := appengine.NewContext(r)

	if v := memcacheGet(c, keyAllKeys); v != nil {
		check(json.Unmarshal(v, &allKeys))
	}

	var alreadyExists bool
	for _, k := range allKeys {
		if k == ap.CacheKey {
			alreadyExists = true
			break
		}
	}

	if !alreadyExists {
		allKeys = append(allKeys, ap.CacheKey)
		allKeysBytes, err := json.Marshal(allKeys)
		check(err)
		memcacheSet(c, keyAllKeys, allKeysBytes)
	}

	memcacheSet(c, ap.CacheKey, []byte(ap.CacheValue))

	t, err := template.New("add-page").Parse(addPageHtml)
	check(err)

	t.Execute(w, ap)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	c := appengine.NewContext(r)

	var value string
	if v := memcacheGet(c, r.FormValue("cache-key")); v != nil {
		value = string(v)
	} else {
		value = "<nil>"
	}

	gp := getPage{
		CacheKey:   r.FormValue("cache-key"),
		CacheValue: value,
	}
	t, err := template.New("list-page").Parse(getPageHtml)
	check(err)

	t.Execute(w, gp)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	var allKeys []string

	c := appengine.NewContext(r)
	if v := memcacheGet(c, keyAllKeys); v != nil {
		check(json.Unmarshal(v, &allKeys))
	}

	key := r.FormValue("cache-key")
	for i, k := range allKeys {
		if k == key {
			allKeys = append(allKeys[:i], allKeys[i+1:]...)

			if allKeysBytes, err := json.Marshal(allKeys); err != nil {
				check(err)
			} else {
				memcacheSet(c, keyAllKeys, allKeysBytes)
			}
		}
	}
	dp := deletePage{
		CacheKey: key,
	}
	t, err := template.New("delete-page").Parse(deletePageHtml)
	check(err)

	t.Execute(w, dp)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	var allKeys []string

	c := appengine.NewContext(r)

	if v := memcacheGet(c, keyAllKeys); v != nil {
		check(json.Unmarshal(v, &allKeys))
	}

	allData := make(map[string]string)
	for _, k := range allKeys {
		if v := memcacheGet(c, k); v != nil {
			allData[k] = string(v)
		}
	}

	page := listPage{
		Data: allData,
	}
	t, err := template.New("list-page").Parse(listPageHtml)
	check(err)

	t.Execute(w, page)
}

// memcacheGet returns an item from memcache or nil if it does not exist. If
// there is an error when retriving the item from memcache, it is a fatal error.
func memcacheGet(c appengine.Context, key string) []byte {
	if item, err := memcache.Get(c, key); err == memcache.ErrCacheMiss {
		return nil
	} else if err != nil {
		c.Errorf("error getting item: %v", err)
		check(err)
	} else {
		return item.Value
	}
	// should not reach here
	return nil
}

// memcacheSet sets a value in memcache. It is a fatal error when the item
// cannot be set.
func memcacheSet(c appengine.Context, key string, value []byte) {
	item := &memcache.Item{
		Key:   key,
		Value: value,
	}
	check(memcache.Set(c, item))
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
