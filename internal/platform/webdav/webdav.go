// Package webdav is the share the readers mount: one WebDAV tree per hub,
// `Ink/<project>/…`, each device logging in with its own token and seeing
// only the projects it is scoped to (ADR 0001, brief §6.3). It listens on the
// Tailscale interface or loopback only.
package webdav

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/webdav"

	"github.com/mkubasz/agentink/internal/platform/api"
)

// Root is the top folder every device sees.
const Root = "Ink"

// DefaultMaxUpload caps a PUT: a scanned, annotated PDF is tens of MB at most.
const DefaultMaxUpload = 64 << 20

// Authenticator checks a device's login.
type Authenticator interface {
	Authenticate(name, token string) (api.Device, bool)
}

// Allowed says whether the device may see a project.
type Allowed func(d api.Device, project string) bool

// Put describes one completed upload.
type Put struct {
	Device  api.Device
	Project string
	// Rel is the path below Ink/<project>/, e.g. "01J….pdf" or "01J….mark".
	Rel string
	// Path is the file on disk.
	Path string
	Size int64
	At   time.Time
}

// Handler serves the tree under dir.
type Handler struct {
	dir       string
	auth      Authenticator
	allowed   Allowed
	onPut     func(Put)
	maxUpload int64
	dav       *webdav.Handler
}

// New returns the handler. onPut may be nil.
func New(dir string, auth Authenticator, allowed Allowed, onPut func(Put), maxUpload int64) (*Handler, error) {
	if err := os.MkdirAll(filepath.Join(dir, Root), 0o700); err != nil {
		return nil, fmt.Errorf("webdav: %w", err)
	}

	if maxUpload <= 0 {
		maxUpload = DefaultMaxUpload
	}

	h := &Handler{dir: dir, auth: auth, allowed: allowed, onPut: onPut, maxUpload: maxUpload}
	h.dav = &webdav.Handler{
		FileSystem: webdav.Dir(dir),
		LockSystem: webdav.NewMemLS(),
	}

	return h, nil
}

// Dir is the tree's root on disk.
func (h *Handler) Dir() string { return h.dir }

// ProjectDir is where a project's pages are published for the readers.
func (h *Handler) ProjectDir(project string) string {
	return filepath.Join(h.dir, Root, project)
}

// ServeHTTP authenticates the device, scopes it to its projects, caps uploads
// and reports completed PUTs.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name, token, ok := r.BasicAuth()
	if !ok || name == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="AgentInk"`)
		http.Error(w, "a device name and token are needed", http.StatusUnauthorized)

		return
	}

	device, ok := h.auth.Authenticate(name, token)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="AgentInk"`)
		http.Error(w, "unknown device or token", http.StatusUnauthorized)

		return
	}

	project, rel, ok := h.scope(device, r.URL.Path)
	if !ok {
		http.Error(w, "this device cannot see that path", http.StatusForbidden)

		return
	}

	if r.Method == http.MethodPut {
		h.put(w, r, device, project, rel)

		return
	}

	if r.Method == http.MethodDelete || r.Method == "MOVE" || r.Method == "COPY" {
		http.Error(w, "readers do not delete or move pages", http.StatusForbidden)

		return
	}

	h.dav.ServeHTTP(w, r)
}

// scope maps a request path to (project, rel) and checks the device's
// projects. The roots "/" and "/Ink/" are listable by everyone so clients can
// browse; a project folder needs the scope.
func (h *Handler) scope(device api.Device, p string) (project, rel string, ok bool) {
	clean := path.Clean("/" + p)
	if clean == "/" || clean == "/"+Root {
		return "", "", true
	}

	parts := strings.SplitN(strings.TrimPrefix(clean, "/"), "/", 3)
	if len(parts) < 2 || parts[0] != Root || parts[1] == "" {
		return "", "", false
	}

	if !h.allowed(device, parts[1]) {
		return "", "", false
	}

	if len(parts) == 3 {
		rel = parts[2]
	}

	return parts[1], rel, true
}

// put stores an upload under the cap, then reports it.
func (h *Handler) put(w http.ResponseWriter, r *http.Request, device api.Device, project, rel string) {
	if rel == "" || strings.HasSuffix(rel, "/") {
		http.Error(w, "upload a file, not a folder", http.StatusBadRequest)

		return
	}

	if r.ContentLength > h.maxUpload {
		http.Error(w, fmt.Sprintf("upload exceeds %d bytes", h.maxUpload), http.StatusRequestEntityTooLarge)

		return
	}

	target := filepath.Join(h.ProjectDir(project), filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	size, err := writeCapped(target, io.LimitReader(r.Body, h.maxUpload+1), h.maxUpload)
	if err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, errTooLarge) {
			code = http.StatusRequestEntityTooLarge
		}

		http.Error(w, err.Error(), code)

		return
	}

	w.WriteHeader(http.StatusCreated)

	if h.onPut != nil {
		h.onPut(Put{Device: device, Project: project, Rel: rel, Path: target, Size: size, At: time.Now()})
	}
}

var errTooLarge = errors.New("webdav: upload exceeds the cap")

// writeCapped writes body to target through a temp file and renames it into
// place, so a reader never sees a half-written file.
func writeCapped(target string, body io.Reader, maxUpload int64) (int64, error) {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".upload-*")
	if err != nil {
		return 0, fmt.Errorf("webdav: %w", err)
	}

	defer os.Remove(tmp.Name()) //nolint:errcheck // gone after the rename, harmless otherwise

	n, err := io.Copy(tmp, body)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		return 0, fmt.Errorf("webdav: write: %w", err)
	}

	if n > maxUpload {
		return 0, errTooLarge
	}

	if err := os.Rename(tmp.Name(), target); err != nil {
		return 0, fmt.Errorf("webdav: %w", err)
	}

	return n, nil
}

// ErrNotTailscale is returned when the listen address is neither a Tailscale
// address nor loopback and the override is not set.
var ErrNotTailscale = errors.New("webdav: refusing to listen outside Tailscale or loopback (brief §7); set --webdav-allow-any to override")

// tailnet is Tailscale's CGNAT range.
var tailnet = net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// Listen binds addr for the readers. The host must be a Tailscale IPv4 or
// loopback unless allowAny; the empty host means "the Tailscale interface".
func Listen(ctx context.Context, addr string, allowAny bool) (net.Listener, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("webdav: listen address %q: %w", addr, err)
	}

	if host == "" {
		host, err = TailscaleIPv4()
		if err != nil {
			return nil, err
		}
	}

	ip := net.ParseIP(host)
	if !allowAny && (ip == nil || (!ip.IsLoopback() && !tailnet.Contains(ip))) {
		return nil, fmt.Errorf("%w: %s", ErrNotTailscale, host)
	}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, fmt.Errorf("webdav: listen %s: %w", addr, err)
	}

	return ln, nil
}

// TailscaleIPv4 finds this machine's address in the tailnet.
func TailscaleIPv4() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("webdav: interfaces: %w", err)
	}

	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil && tailnet.Contains(ipn.IP) {
			return ipn.IP.String(), nil
		}
	}

	return "", errors.New("webdav: no Tailscale interface found; set the listen address to 127.0.0.1:<port> and use `tailscale serve`, or --webdav-listen")
}

// EqualToken compares two tokens in constant time (for tests and tooling).
func EqualToken(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
