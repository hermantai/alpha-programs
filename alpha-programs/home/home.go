package home

import (
	"fmt"
	"net/http"
)

func init() {
	http.HandleFunc("/", homeHandler)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `
<!DOCTYPE html>
<html>
<head>
	<title>Alpha programs</title>
</head>
<body>
<p><a href="/mycache">My Cache</a></p>
<p><a href="/lissajous/lissajous.gif">Lissajous</a></p>
</body>
</html>
`)
}
