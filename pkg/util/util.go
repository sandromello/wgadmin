package util

import (
	"crypto/rand"
	"fmt"
	"math"
	"time"
)

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(n int) (string, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return "", err
	}
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), err
}

// RoundTime round time based on a resolution (r) from a given a duration (d)
func RoundTime(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}

// GetDeltaDuration computes the delta between two string RFC3339 dates
func GetDeltaDuration(startTime, endTime string) string {
	start, _ := time.Parse(time.RFC3339, startTime)
	end, _ := time.Parse(time.RFC3339, endTime)
	delta := end.Sub(start)
	var d time.Duration
	if endTime != "" {
		d = RoundTime(delta, time.Second)
	} else {
		d = RoundTime(time.Since(start), time.Second)
	}
	switch {
	case d.Hours() >= 24: // day resolution
		return fmt.Sprintf("%.fd", math.Floor(d.Hours()/24))
	case d.Hours() >= 8760: // year resolution
		return fmt.Sprintf("%.fd", math.Floor(d.Hours()/8760))
	}
	return d.String()
}
