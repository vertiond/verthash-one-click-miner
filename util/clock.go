package util

import (
	"fmt"
	"net/http"
	"time"
)

var clockClient = &http.Client{Timeout: 10 * time.Second}

// ClockSkew reports how far the local clock is from the server at url, as
// local time minus server time, from the Date header of a HEAD request. Half
// the round trip is added to the server's time. The header has one-second
// resolution, so small skews read as zero or a second either way.
func ClockSkew(url string) (time.Duration, error) {
	start := time.Now()
	resp, err := clockClient.Head(url)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	end := time.Now()

	date := resp.Header.Get("Date")
	if date == "" {
		return 0, fmt.Errorf("no Date header from %s", url)
	}
	server, err := http.ParseTime(date)
	if err != nil {
		return 0, fmt.Errorf("unreadable Date header from %s: %w", url, err)
	}
	server = server.Add(end.Sub(start) / 2)
	return end.Sub(server), nil
}
