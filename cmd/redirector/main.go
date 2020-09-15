package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
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
		log.Printf("request: method=%s ip=%s referer=%s params=%s", req.Method, req.Header.Get("x-forwarded-for"), req.Header.Get(hdrReferer), req.URL.RawQuery)
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
	if req.Method == http.MethodPost {
		manualRedirect(w, req)
		return
	}
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "method %s not allowed", req.Method)
		return
	}

	repoParam := req.URL.Query().Get(paramRepo)
	referer := req.Header.Get(hdrReferer)

	// TODO(ahmetb): remove once https://github.community/t/chrome-85-breaks-referer/130039 is fixed
	if referer == "https://github.com/" && repoParam != "" {
		showRedirectForm(w, req)
		return
	}

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
	doRedirect(w, repo, req.URL.Query())
}

func doRedirect(w http.ResponseWriter, r repoRef, overrides url.Values) {
	target := prepURL(r, overrides)
	w.Header().Set("location", target)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// TODO(ahmetb): remove once https://github.community/t/chrome-85-breaks-referer/130039 is fixed
func showRedirectForm(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
	<head>
		<title>Cloud Run Button</title>
		<style>
			html {
				font-family: '-apple-system','BlinkMacSystemFont','segoe ui',Roboto,'helvetica neue',Arial, sans-serif, 'apple color emoji', 'segoe ui emoji', 'segoe ui symbol';
			}
			body {
				margin: 0;
			}
			html, input {
				font-size: 120%%;
			}
			.container {
				margin: 5em auto;
				max-width: 768px;
			}
		</style>
	</head>
	<body>
		<div class="container">
		<h1>You’re almost there!</h1>
		<p>
			Unfortunately with the new Chrome 85, GitHub temporarily breaks our
			ability to determine which GitHub repository you came from. (You can
			<a href="https://github.community/t/chrome-85-breaks-referer/130039"
			rel="nofolow">help us ask GitHub</a> to fix it!)
		</p>
		<p>
			Please provide the URL of the previous page you came from:
		</p>
		<form action="/" method="POST">
			<input type="text" name="url" placeholder="https://github.com/..."
				size="32"/>
			<input type="hidden" name="orig_query" value="%s"/>
			<input type="submit" value="Deploy!"/>
		</form>
		</div>
	</body>
</html>`, r.URL.Query().Encode())
}

// TODO(ahmetb): remove once https://github.community/t/chrome-85-breaks-referer/130039 is fixed
func manualRedirect(w http.ResponseWriter, req *http.Request) {
	refURL := req.FormValue("url")
	origQuery, err := url.ParseQuery(req.FormValue("orig_query"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, errors.Wrapf(err, "failed to parse orig_query=%q: %v", origQuery, err).Error())
		return
	}
	repo, err := parseReferer(refURL, availableExtractors)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, errors.Wrapf(err, "failed to parse url into a github repository: %s", refURL).Error())
		return
	}
	doRedirect(w, repo, origQuery)
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
