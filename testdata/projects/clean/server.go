// Clean file with no security issues — used for integration testing.
package main

import (
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	origin := os.Getenv("ALLOWED_ORIGIN")
	w.Header().Set("Access-Control-Allow-Origin", origin)

	id := r.URL.Query().Get("id")
	// Parameterized query — no SEC-201
	_ = id
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
