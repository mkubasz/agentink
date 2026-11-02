// Package domain is the ink context: the readers, their access, what they
// send back and what the worker made of it (ADR 0001).
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// Event types the context emits about devices.
const (
	EventDeviceAdded   = "DeviceAdded"
	EventDeviceRevoked = "DeviceRevoked"
)

// HubProject is the project slug under which hub-wide events (devices) are
// logged: they belong to no repository.
const HubProject = "hub"

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// Errors of the device lifecycle.
var (
	ErrBadName   = errors.New("ink: device name must be lowercase letters, digits and hyphens")
	ErrNotFound  = errors.New("ink: no such device")
	ErrExists    = errors.New("ink: a device with that name exists")
	ErrNoProject = errors.New("ink: a device needs at least one project")
)

// Added is the payload of DeviceAdded. The token itself is never logged:
// only its hash.
type Added struct {
	Name      string   `json:"name"`
	Profile   string   `json:"profile"`
	TokenHash string   `json:"token_hash"`
	Projects  []string `json:"projects"`
}

// Validate checks a registration.
func (a *Added) Validate() error {
	if !nameRe.MatchString(a.Name) {
		return fmt.Errorf("%w: %q", ErrBadName, a.Name)
	}

	if len(a.Projects) == 0 {
		return ErrNoProject
	}

	if a.Profile == "" {
		a.Profile = "supernote-manta"
	}

	sort.Strings(a.Projects)

	return nil
}

// NewToken returns a fresh device token and its hash. The token is shown to
// the owner once, typed into the reader's WebDAV client as the password, and
// never stored in clear.
func NewToken() (token, hash string, err error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("ink: token: %w", err)
	}

	token = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)

	return token, HashToken(token), nil
}

// HashToken is sha256 in hex: what the log and the projection keep.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

// Devices is the projection of every device the hub knows.
type Devices struct {
	mu   sync.RWMutex
	byID map[string]api.Device
	hash map[string]string // name -> token hash
}

// NewDevices returns an empty projection.
func NewDevices() *Devices { return &Devices{byID: map[string]api.Device{}, hash: map[string]string{}} }

// Apply folds DeviceAdded and DeviceRevoked.
func (ds *Devices) Apply(e eventlog.Envelope) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	switch e.Type {
	case EventDeviceAdded:
		var a Added
		if err := json.Unmarshal(e.Data, &a); err != nil {
			return fmt.Errorf("ink: bad %s in %s: %w", e.Type, e.ID, err)
		}

		ds.byID[a.Name] = api.Device{Name: a.Name, Profile: a.Profile, Projects: a.Projects, AddedAt: e.At}
		ds.hash[a.Name] = a.TokenHash
	case EventDeviceRevoked:
		var r struct {
			Name string `json:"name"`
		}

		if err := json.Unmarshal(e.Data, &r); err != nil {
			return fmt.Errorf("ink: bad %s in %s: %w", e.Type, e.ID, err)
		}

		if d, ok := ds.byID[r.Name]; ok {
			at := e.At
			d.RevokedAt = &at
			ds.byID[r.Name] = d
			delete(ds.hash, r.Name)
		}
	}

	return nil
}

// Get returns one device.
func (ds *Devices) Get(name string) (api.Device, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	d, ok := ds.byID[name]

	return d, ok
}

// List returns every device, revoked ones included, sorted by name.
func (ds *Devices) List() []api.Device {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	out := make([]api.Device, 0, len(ds.byID))
	for _, d := range ds.byID {
		out = append(out, d)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out
}

// Authenticate checks a WebDAV login: the device name and its token. A
// revoked device never authenticates.
func (ds *Devices) Authenticate(name, token string) (api.Device, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	want, ok := ds.hash[name]
	if !ok {
		return api.Device{}, false
	}

	got := HashToken(token)
	if subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		return api.Device{}, false
	}

	d := ds.byID[name]
	if d.RevokedAt != nil {
		return api.Device{}, false
	}

	return d, true
}

// Allowed says whether the device may see a project's folder.
func Allowed(d api.Device, project string) bool {
	for _, p := range d.Projects {
		if p == project {
			return true
		}
	}

	return false
}

// Now is the clock the slices use; tests replace it.
var Now = time.Now
