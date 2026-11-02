package webdav_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/webdav"
)

type auth map[string]api.Device

func (a auth) Authenticate(name, token string) (api.Device, bool) {
	d, ok := a[name]
	if !ok || token != "secret-"+name {
		return api.Device{}, false
	}

	return d, true
}

func allowed(d api.Device, project string) bool {
	for _, p := range d.Projects {
		if p == project {
			return true
		}
	}

	return false
}

func newServer(t *testing.T, onPut func(webdav.Put)) (*httptest.Server, string) {
	t.Helper()

	dir := t.TempDir()
	devices := auth{"manta": {Name: "manta", Projects: []string{"shop"}}}

	h, err := webdav.New(dir, devices, allowed, onPut, 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return srv, dir
}

// do performs one request and returns the status and the body, closed.
func do(t *testing.T, srv *httptest.Server, method, path, user, pass string, body io.Reader) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), method, srv.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}

	if user != "" {
		req.SetBasicAuth(user, pass)
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close() //nolint:errcheck // test

	raw, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, string(raw)
}

func TestWebDAV_DeviceTokenScopesTree(t *testing.T) {
	t.Parallel()

	var puts []webdav.Put

	srv, dir := newServer(t, func(p webdav.Put) { puts = append(puts, p) })

	_ = os.MkdirAll(filepath.Join(dir, "Ink", "shop"), 0o750)
	_ = os.MkdirAll(filepath.Join(dir, "Ink", "other"), 0o750)
	_ = os.WriteFile(filepath.Join(dir, "Ink", "shop", "page.pdf"), []byte("%PDF"), 0o600)

	cases := []struct {
		name, method, path, user, pass string
		want                           int
	}{
		{"no auth", http.MethodGet, "/Ink/shop/page.pdf", "", "", http.StatusUnauthorized},
		{"bad token", http.MethodGet, "/Ink/shop/page.pdf", "manta", "nope", http.StatusUnauthorized},
		{"foreign project", http.MethodGet, "/Ink/other/", "manta", "secret-manta", http.StatusForbidden},
		{"browse root", "PROPFIND", "/Ink/", "manta", "secret-manta", http.StatusMultiStatus},
		{"delete", http.MethodDelete, "/Ink/shop/page.pdf", "manta", "secret-manta", http.StatusForbidden},
	}

	for _, tc := range cases {
		if status, _ := do(t, srv, tc.method, tc.path, tc.user, tc.pass, nil); status != tc.want {
			t.Errorf("%s: status %d, want %d", tc.name, status, tc.want)
		}
	}

	if status, body := do(t, srv, http.MethodGet, "/Ink/shop/page.pdf", "manta", "secret-manta", nil); status != http.StatusOK || body != "%PDF" {
		t.Fatalf("read own project: %d %q", status, body)
	}

	assertPut(t, srv, dir, &puts)
}

func assertPut(t *testing.T, srv *httptest.Server, dir string, puts *[]webdav.Put) {
	t.Helper()

	if status, _ := do(t, srv, http.MethodPut, "/Ink/shop/page.mark", "manta", "secret-manta", strings.NewReader("ink!")); status != http.StatusCreated {
		t.Fatalf("put: %d", status)
	}

	got, err := os.ReadFile(filepath.Join(dir, "Ink", "shop", "page.mark")) //nolint:gosec // test dir
	if err != nil || string(got) != "ink!" {
		t.Fatalf("upload on disk = %q (%v)", got, err)
	}

	p := *puts
	if len(p) != 1 || p[0].Project != "shop" || p[0].Rel != "page.mark" || p[0].Size != 4 || p[0].Device.Name != "manta" {
		t.Fatalf("puts = %+v", p)
	}
}

func TestWebDAV_RejectsOversizedPut(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t, nil)
	big := strings.NewReader(strings.Repeat("x", 1<<20+1))

	if status, _ := do(t, srv, http.MethodPut, "/Ink/shop/big.pdf", "manta", "secret-manta", big); status != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized put: %d", status)
	}
}

func TestListener_RefusesNonTailscale(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if _, err := webdav.Listen(ctx, "0.0.0.0:0", false); !errors.Is(err, webdav.ErrNotTailscale) {
		t.Fatalf("0.0.0.0: %v", err)
	}

	if _, err := webdav.Listen(ctx, "192.168.1.10:0", false); !errors.Is(err, webdav.ErrNotTailscale) {
		t.Fatalf("lan: %v", err)
	}

	ln, err := webdav.Listen(ctx, "127.0.0.1:0", false)
	if err != nil {
		t.Fatalf("loopback: %v", err)
	}

	_ = ln.Close()

	ln, err = webdav.Listen(ctx, "0.0.0.0:0", true)
	if err != nil {
		t.Fatalf("override: %v", err)
	}

	_ = ln.Close()
}
