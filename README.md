# go-ip-ratelimit

A simple in-memory sliding window rate limiter for Go. Zero dependencies beyond the standard library.

Extracted from the [AllSource Control Plane](https://github.com/all-source-os/all-source).

## Install

```bash
go get github.com/all-source-os/go-ip-ratelimit
```

## Usage

```go
package main

import (
	"net/http"
	"time"

	ratelimit "github.com/all-source-os/go-ip-ratelimit"
)

func main() {
	// 5 requests per IP per hour
	limiter := ratelimit.New(5, time.Hour)
	defer limiter.Stop()

	http.HandleFunc("/api/signup", func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if !limiter.Allow(ip) {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", limiter.RetryAfter(ip)))
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("ok"))
	})

	http.ListenAndServe(":8080", nil)
}
```

## API

| Method | Description |
|---|---|
| `New(limit, window)` | Create a limiter. Starts background cleanup goroutine. |
| `Allow(key) bool` | Check + record a request. Returns false if over limit. |
| `RetryAfter(key) int` | Seconds until the oldest request expires. For Retry-After header. |
| `Remaining(key) int` | How many requests left in the current window. |
| `Reset(key)` | Clear all recorded requests for a key. |
| `Stop()` | Stop the background cleanup goroutine. |

## License

MIT — see [LICENSE](LICENSE).
