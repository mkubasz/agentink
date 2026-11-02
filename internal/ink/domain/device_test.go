package domain_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mkubasz/agentink/internal/ink/domain"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

func added(t *testing.T, name, hash string, at time.Time) eventlog.Envelope {
	t.Helper()

	data, _ := json.Marshal(domain.Added{Name: name, Profile: "supernote-manta", TokenHash: hash, Projects: []string{"shop"}})

	return eventlog.Envelope{ID: eventlog.NewID(at), Type: domain.EventDeviceAdded, At: at, Project: domain.HubProject, Subject: "device/" + name, Rev: 1, Data: data}
}

func TestDevices_TokenAuthenticatesUntilRevoked(t *testing.T) {
	t.Parallel()

	token, hash, err := domain.NewToken()
	if err != nil || len(token) < 30 || hash != domain.HashToken(token) {
		t.Fatalf("token %q hash %q err %v", token, hash, err)
	}

	ds := domain.NewDevices()
	at := time.Now()

	if err := ds.Apply(added(t, "manta", hash, at)); err != nil {
		t.Fatal(err)
	}

	assertAuth(t, ds, token)

	revoked, _ := json.Marshal(map[string]string{"name": "manta"})
	if err := ds.Apply(eventlog.Envelope{ID: eventlog.NewID(at), Type: domain.EventDeviceRevoked, At: at, Project: domain.HubProject, Subject: "device/manta", Rev: 1, Data: revoked}); err != nil {
		t.Fatal(err)
	}

	if _, ok := ds.Authenticate("manta", token); ok {
		t.Fatal("revoked device still authenticates")
	}

	if list := ds.List(); len(list) != 1 || list[0].RevokedAt == nil {
		t.Fatalf("list = %+v", list)
	}
}

func assertAuth(t *testing.T, ds *domain.Devices, token string) {
	t.Helper()

	d, ok := ds.Authenticate("manta", token)
	if !ok || d.Name != "manta" || !domain.Allowed(d, "shop") || domain.Allowed(d, "other") {
		t.Fatalf("authenticate: %+v %v", d, ok)
	}

	if _, ok := ds.Authenticate("manta", "wrong"); ok {
		t.Fatal("wrong token accepted")
	}

	if _, ok := ds.Authenticate("nobody", token); ok {
		t.Fatal("unknown device accepted")
	}
}

func TestAdded_Validate(t *testing.T) {
	t.Parallel()

	bad := domain.Added{Name: "My Manta", Projects: []string{"shop"}}
	if err := bad.Validate(); !errors.Is(err, domain.ErrBadName) {
		t.Fatalf("err = %v", err)
	}

	none := domain.Added{Name: "manta"}
	if err := none.Validate(); !errors.Is(err, domain.ErrNoProject) {
		t.Fatalf("err = %v", err)
	}

	ok := domain.Added{Name: "manta", Projects: []string{"zeta", "alpha"}}
	if err := ok.Validate(); err != nil || ok.Profile != "supernote-manta" || ok.Projects[0] != "alpha" {
		t.Fatalf("defaults: %+v %v", ok, err)
	}
}
