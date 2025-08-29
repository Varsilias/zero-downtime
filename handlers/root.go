package handlers

import "net/http"

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello Zero Downtime"))
}
