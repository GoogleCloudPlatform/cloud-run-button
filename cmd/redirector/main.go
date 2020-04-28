package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

const (
	hdrReferer = "Referer"
)

func main() {
	log.SetFlags(log.Lmicroseconds)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	listenAddr := ":" + port

	// TODO(ahmetb): extract server logic to NewServer() to unit test it
	mux := http.NewServeMux()
	mux.HandleFunc("/", withLogging(redirect))
	mux.HandleFunc("/button.svg", staticRedirect("https://storage.googleapis.com/cloudrun/button.svg", http.StatusMovedPermanently))
	mux.HandleFunc("/button.png", staticRedirect("https://storage.googleapis.com/cloudrun/button.png", http.StatusMovedPermanently))

	err := http.ListenAndServe(listenAddr, mux)
	if err == http.ErrServerClosed {
		log.Printf("server successfully closed")
	} else if err != nil {
		log.Fatal(err)
	}
}

func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Printf("request: method=%s ip=%s referer=%s params=%s", req.Method, req.RemoteAddr, req.Header.Get(hdrReferer), req.URL.RawQuery)
		ww := &respRecorder{w: w}
		next(ww, req)
		log.Printf("response: status=%d location=%s", ww.status, w.Header().Get("location"))
	}
}

func staticRedirect(url string, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("location", url)
		w.WriteHeader(code)
	}
}

func redirect(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "method %s not allowed", req.Method)
		return
	}

	repoParam := req.URL.Query().Get(paramRepo)
	referer := req.Header.Get(hdrReferer)

	var repo repoRef
	if repoParam != "" {
		repo = customRepoRef{req.URL.Query()}
	} else {
		if referer == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Cannot infer which repository to deploy (%s header was not present).\n", hdrReferer)
			fmt.Fprintln(w, "Go back, and click the 'Run on Google Cloud' button directly from the repository page.")
			return
		}
		r, err := parseReferer(referer, availableExtractors)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, errors.Wrapf(err, "failed to parse %s header", hdrReferer).Error())
			return
		}
		repo = r
	}

	target := prepURL(repo, req.URL.Query())
	w.Header().Set("location", target)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

type respRecorder struct {
	w      http.ResponseWriter
	status int
}

func (rr *respRecorder) Header() http.Header         { return rr.w.Header() }
func (rr *respRecorder) Write(p []byte) (int, error) { return rr.w.Write(p) }
func (rr *respRecorder) WriteHeader(statusCode int) {
	rr.status = statusCode
	rr.w.WriteHeader(statusCode)
}
