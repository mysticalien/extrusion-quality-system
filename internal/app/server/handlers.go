package server

import (
	"fmt"
	nethttp "net/http"
)

func homeHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Path != "/" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	nethttp.ServeFile(w, r, "web/index.html")
}

func healthHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_, _ = fmt.Fprintln(w, `{"status":"ok"}`)
}
