// Intentionally vulnerable file for integration testing.
package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// SEC-301: Wildcard CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// SEC-201: SQL via fmt.Sprintf
	id := r.URL.Query().Get("id")
	query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", id)
	_ = query

	// SEC-108: Database URL with credentials
	dsn := "postgres://admin:s3cr3tpassword@prod-db.example.com/appdb"
	_ = dsn
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
