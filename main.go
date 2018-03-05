package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
	"unicode"
)

const (
	// DefaultDownloadSize is the default size for responses to GET requests
	DefaultDownloadSize = "1M"
)

// ParseSize converts a size string into a number of bytes. A size string is an
// integer with an optional suffix: 'B' for bytes, 'K' for kilobytes, 'M' for
// megabytes, or 'G' for gigabytes. If no suffix is provided, bytes is assumed.
func ParseSize(s string) (bytes int64, err error) {
	if s == "" {
		return
	}

	var unit int64
	switch unicode.ToLower(rune(s[len(s)-1])) {
	case 'b':
		unit = 1
	case 'k':
		unit = 1024
	case 'm':
		unit = 1024 * 1024
	case 'g':
		unit = 1024 * 1024 * 1024
	}

	if unit > 0 {
		s = s[:len(s)-1]
	} else {
		unit = 1
	}

	bytes, err = strconv.ParseInt(s, 10, 64)
	bytes *= unit
	return
}

// Handler accepts GET and PUT request on all paths. GET requests response with
// a random binary file, while PUT requests discard all received data.
type Handler struct{}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("[REQUEST] %s %s", req.Method, req.URL)

	switch req.Method {
	case http.MethodGet:
		h.ServeDownload(w, req)
	case http.MethodPut:
		h.ServeUpload(w, req)
	default:
		code := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(code), code)
	}
}

// ServeDownload responds with a random binary file of the requested size.
func (h *Handler) ServeDownload(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	size := query.Get("size")
	if size == "" {
		size = DefaultDownloadSize
	}

	bytes, err := ParseSize(size)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	src := rand.NewSource(time.Now().Unix())
	r := rand.New(src)

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Length", strconv.FormatInt(bytes, 10))

	w.WriteHeader(http.StatusOK)
	if n, err := io.CopyN(w, r, bytes); err != nil {
		log.Printf("[ERROR] incomplete write: wanted = %d, wrote = %d: %v\n", bytes, n, err)
	}
}

// ServeUpload reads and discards all data in the request body.
func (h *Handler) ServeUpload(w http.ResponseWriter, req *http.Request) {
	n, err := io.Copy(ioutil.Discard, req.Body)
	if err != nil {
		msg := fmt.Sprintf("incomplete read: wanted = %d, wrote = %d: %v\n", req.ContentLength, n, err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

var opts struct {
	Address string
}

func defineAndParseFlags() {
	flag.StringVar(&opts.Address, "address", "127.0.0.1:8000", "the address to listen on")
	flag.Parse()
}

func main() {
	defineAndParseFlags()

	log.Printf("Starting server on %s\n", opts.Address)
	log.Fatal(http.ListenAndServe(opts.Address, &Handler{}))
}
