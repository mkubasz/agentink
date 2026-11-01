package eventlog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// ErrCorrupt is returned when a line that is not the last one fails to parse.
var ErrCorrupt = errors.New("eventlog: corrupt line")

// maxLine is the longest line Read accepts: 4 MiB, far above any envelope.
const maxLine = 4 << 20

// Log is one project's append-only file. Append is safe for concurrent use;
// every append is one write followed by fsync, so a crash leaves at most one
// truncated last line, which Read skips and reports.
type Log struct {
	path     string
	mu       sync.Mutex
	f        *os.File
	repaired int64
}

// Open opens or creates the log at path, creating parent directories. A file
// that does not end with a newline holds a torn last write; Open truncates it
// back to the last complete line so the next Append starts a clean line, and
// Repaired reports how many bytes were cut.
func Open(path string) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("eventlog: open %s: %w", path, err)
	}

	repaired, err := repairTail(path)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // path comes from the hub configuration, not from a request
	if err != nil {
		return nil, fmt.Errorf("eventlog: open %s: %w", path, err)
	}

	return &Log{path: path, f: f, repaired: repaired}, nil
}

// repairTail cuts a trailing partial line and returns how many bytes it cut.
func repairTail(path string) (int64, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0o600) //nolint:gosec // path comes from the hub configuration, not from a request
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("eventlog: repair %s: %w", path, err)
	}

	defer f.Close() //nolint:errcheck // best effort after the truncate below

	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("eventlog: repair %s: %w", path, err)
	}

	size := info.Size()
	if size == 0 {
		return 0, nil
	}

	// Read the tail in one chunk; a torn line is never longer than maxLine.
	from := max(size-maxLine, 0)
	tail := make([]byte, size-from)

	if _, err := f.ReadAt(tail, from); err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("eventlog: repair %s: %w", path, err)
	}

	if tail[len(tail)-1] == '\n' {
		return 0, nil
	}

	keep := from + int64(bytes.LastIndexByte(tail, '\n')) + 1
	if err := f.Truncate(keep); err != nil {
		return 0, fmt.Errorf("eventlog: repair %s: %w", path, err)
	}

	if err := f.Sync(); err != nil {
		return 0, fmt.Errorf("eventlog: repair %s: %w", path, err)
	}

	return size - keep, nil
}

// Path is the file the log writes to.
func (l *Log) Path() string { return l.path }

// Repaired is the number of bytes of a torn last line Open cut, or 0.
func (l *Log) Repaired() int64 { return l.repaired }

// Append validates e, writes it as one line and fsyncs before returning.
func (l *Log) Append(e Envelope) error {
	if err := e.Validate(); err != nil {
		return err
	}

	b, err := e.MarshalJSON()
	if err != nil {
		return err
	}

	b = append(b, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.f.Write(b); err != nil {
		return fmt.Errorf("eventlog: append %s: %w", e.ID, err)
	}

	if err := l.f.Sync(); err != nil {
		return fmt.Errorf("eventlog: fsync after %s: %w", e.ID, err)
	}

	return nil
}

// Close closes the file; the log must not be used afterwards.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.f.Close(); err != nil {
		return fmt.Errorf("eventlog: close %s: %w", l.path, err)
	}

	return nil
}

// ReadResult is what Read found besides the envelopes.
type ReadResult struct {
	// Truncated is set when the last line was incomplete and skipped.
	Truncated bool
}

// Read replays every envelope in path, in order, to fn. A last line without a
// newline or that fails to parse is skipped and reported through the result;
// any other unparsable line is ErrCorrupt, because nothing but a crash
// mid-write can produce it and the log must not be trusted past it.
func Read(path string, fn func(Envelope) error) (ReadResult, error) {
	var res ReadResult

	data, err := os.ReadFile(path) //nolint:gosec // path comes from the hub configuration, not from a request
	if errors.Is(err, os.ErrNotExist) {
		return res, nil
	}

	if err != nil {
		return res, fmt.Errorf("eventlog: read %s: %w", path, err)
	}

	terminated := len(data) > 0 && data[len(data)-1] == '\n'
	lines := bytes.Split(data, []byte{'\n'})

	for i, raw := range lines {
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}

		last := i == len(lines)-1 || (!terminated && i == len(lines)-2 && len(bytes.TrimSpace(lines[len(lines)-1])) == 0)

		e, perr := parseLine(raw)
		if perr != nil {
			if last && !terminated {
				res.Truncated = true

				return res, nil
			}

			return res, fmt.Errorf("%w: %s:%d: %w", ErrCorrupt, path, i+1, perr)
		}

		if err := fn(e); err != nil {
			return res, err
		}
	}

	return res, nil
}

func parseLine(raw []byte) (Envelope, error) {
	var e Envelope

	if len(raw) > maxLine {
		return e, fmt.Errorf("line exceeds %d bytes", maxLine)
	}

	if err := e.UnmarshalJSON(raw); err != nil {
		return e, err
	}

	if err := e.Validate(); err != nil {
		return e, err
	}

	return e, nil
}
