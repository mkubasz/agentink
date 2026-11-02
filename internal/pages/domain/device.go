// Package domain is the pages context: what a question looks like on paper —
// device profiles, page types, their fields and zones (ADR 0001, ADR 0006).
package domain

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// ErrUnknownDevice is returned for a profile name the hub does not know.
var ErrUnknownDevice = errors.New("pages: unknown device profile")

// Device is a reader's screen: a page is rendered at exactly this size so
// the PDF maps 1:1 to the display and zone coordinates need no scaling.
type Device struct {
	Name     string
	WidthPx  int
	HeightPx int
	PPI      float64
	Format   string // pdf or png
}

// SizeMM is the page size in millimetres.
func (d Device) SizeMM() (w, h float64) {
	return round2(float64(d.WidthPx) / d.PPI * 25.4), round2(float64(d.HeightPx) / d.PPI * 25.4)
}

// DefaultDevice is used when a project names none: the owner's Supernote
// Manta (2026-11-02).
const DefaultDevice = "supernote-manta"

// Sizes are portrait, in the panel's grayscale resolution.
var profiles = map[string]Device{
	"supernote-a5x":   {Name: "supernote-a5x", WidthPx: 1404, HeightPx: 1872, PPI: 226, Format: "pdf"},
	"supernote-a6x":   {Name: "supernote-a6x", WidthPx: 1404, HeightPx: 1872, PPI: 300, Format: "pdf"},
	"supernote-nomad": {Name: "supernote-nomad", WidthPx: 1404, HeightPx: 1872, PPI: 300, Format: "pdf"},
	"supernote-manta": {Name: "supernote-manta", WidthPx: 1920, HeightPx: 2560, PPI: 300, Format: "pdf"},
	"boox-note-air":   {Name: "boox-note-air", WidthPx: 1404, HeightPx: 1872, PPI: 227, Format: "pdf"},
	// 13.3" Kaleido 3: 3200×2400 grayscale at 300 ppi (1600×1200 in colour); the owner's Boox.
	"boox-tab-x-c": {Name: "boox-tab-x-c", WidthPx: 2400, HeightPx: 3200, PPI: 300, Format: "pdf"},
}

// Lookup returns a device profile by name.
func Lookup(name string) (Device, error) {
	if name == "" {
		name = DefaultDevice
	}

	d, ok := profiles[name]
	if !ok {
		return Device{}, fmt.Errorf("%w: %q", ErrUnknownDevice, name)
	}

	return d, nil
}

// Devices lists the profile names, sorted.
func Devices() []string {
	out := make([]string, 0, len(profiles))
	for n := range profiles {
		out = append(out, n)
	}

	sort.Strings(out)

	return out
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
