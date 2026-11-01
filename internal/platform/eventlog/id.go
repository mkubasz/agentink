package eventlog

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	idMu      sync.Mutex
	idEntropy = ulid.Monotonic(rand.Reader, 0)
)

// NewID returns a new ULID: 26 uppercase Crockford base32 characters,
// time-ordered, monotonic within a millisecond (ADR 0005).
func NewID(now time.Time) string {
	idMu.Lock()
	defer idMu.Unlock()

	return ulid.MustNew(ulid.Timestamp(now), idEntropy).String()
}
