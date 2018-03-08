package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
	"unicode"

	"golang.org/x/time/rate"
)

const (
	// DefaultDownloadSize is the default size for responses to GET requests.
	DefaultDownloadSize int64 = 1024 * 1024
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

// Reader is an io.Reader that implements a rate limit on reads. The rate limit
// holds over the duration of all read operations, but may be exceeded at any
// given instant.
type Reader struct {
	r       io.Reader
	limiter *rate.Limiter
}

// NewReader creates a new rate-limited reader. The limit is in bytes/second.
func NewReader(r io.Reader, limit rate.Limit) *Reader {
	return &Reader{
		r:       r,
		limiter: rate.NewLimiter(limit, burstFromLimit(limit)),
	}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if n <= 0 {
		return
	}

	waitErr := r.wait(n)
	if err == nil {
		err = waitErr
	}
	return
}

func (r *Reader) wait(n int) error {
	burst := r.limiter.Burst()
	for n > 0 {
		waitN := min(n, burst)
		n -= waitN

		err := r.limiter.WaitN(context.TODO(), waitN)
		if err != nil {
			return err
		}
	}
	return nil
}

func burstFromLimit(limit rate.Limit) int {
	ceil := math.Ceil(float64(limit))
	if ceil < float64(math.MaxInt32) {
		return int(ceil)
	}
	return math.MaxInt32
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
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
	size, err := getSize(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("size: %v", err), http.StatusBadRequest)
		return
	}

	limit, err := getLimit(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("rate: %v", err), http.StatusBadRequest)
		return
	}

	preDelay, resDelay, err := getDelays(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	src := rand.NewSource(time.Now().Unix())
	r := NewReader(rand.New(src), limit)

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Length", strconv.FormatInt(size, 10))

	time.Sleep(preDelay + resDelay)

	w.WriteHeader(http.StatusOK)
	if n, err := io.CopyN(w, r, size); err != nil {
		log.Printf("[ERROR] incomplete write: wanted = %d, wrote = %d: %v\n", size, n, err)
	}
}

// ServeUpload reads and discards all data in the request body.
func (h *Handler) ServeUpload(w http.ResponseWriter, req *http.Request) {
	limit, err := getLimit(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("rate: %v", err), http.StatusBadRequest)
		return
	}

	preDelay, resDelay, err := getDelays(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	r := NewReader(req.Body, limit)

	time.Sleep(preDelay)

	n, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		msg := fmt.Sprintf("incomplete read: wanted = %d, wrote = %d: %v\n", req.ContentLength, n, err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	time.Sleep(resDelay)

	w.WriteHeader(http.StatusCreated)
}

func getLimit(req *http.Request) (rate.Limit, error) {
	rateParam := req.URL.Query().Get("rate")
	if rateParam != "" {
		bytes, err := ParseSize(rateParam)
		if err != nil {
			return 0, err
		}
		return rate.Limit(bytes), nil
	}
	return rate.Inf, nil
}

func getSize(req *http.Request) (int64, error) {
	sizeParam := req.URL.Query().Get("size")
	if sizeParam != "" {
		size, err := ParseSize(sizeParam)
		if err != nil {
			return 0, err
		}
		return size, nil
	}
	return DefaultDownloadSize, nil
}

func getDelays(req *http.Request) (pre time.Duration, res time.Duration, err error) {
	parse := func(name string) (d time.Duration, err error) {
		param := req.URL.Query().Get(name)
		if param != "" {
			d, err = time.ParseDuration(param)
			if err != nil {
				err = fmt.Errorf("%s: %v", name, err)
			}
		}
		return
	}

	pre, err = parse("delayPre")
	if err != nil {
		return
	}

	res, err = parse("delayRes")
	return
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
