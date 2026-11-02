package renderpage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mkubasz/agentink/contracts"
	"github.com/mkubasz/agentink/internal/pages/domain"
	"github.com/mkubasz/agentink/internal/pages/renderpage"
)

const (
	questionID = "01JBQ4P9V7G3ZK8XW2M5N6C7D8"
	pageID     = "01JBQ4PB2D6G0J3K4M5N6P7Q8R"
)

var fixedNow = time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)

// typst returns a runner or skips the test when the binary is missing.
func typst(t *testing.T) *renderpage.Typst {
	t.Helper()

	tp, err := renderpage.NewTypst(context.Background(), "", t.TempDir())
	if err != nil {
		if errors.Is(err, renderpage.ErrTypst) {
			t.Skipf("typst not available: %v (install Typst %s.x, ADR 0006)", err, renderpage.Pinned)
		}

		t.Fatal(err)
	}

	return tp
}

func decisionQuestion(t *testing.T) json.RawMessage {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "contracts", "examples", "question", "decision.json"))
	if err != nil {
		t.Fatal(err)
	}

	return raw
}

func TestRenderDecision_Golden(t *testing.T) {
	t.Parallel()

	tp := typst(t)
	device, _ := domain.Lookup(domain.DefaultDevice)
	h := renderpage.New(nil, tp, t.TempDir(), device, func() time.Time { return fixedNow })
	pageDir := filepath.Join(t.TempDir(), "page")

	rendered, err := h.Render(context.Background(), decisionQuestion(t), questionID, 1, pageID, pageDir)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if rendered.PDF != "page.pdf" || rendered.Zones != "zones.json" || rendered.Device != domain.DefaultDevice {
		t.Fatalf("rendered = %+v", rendered)
	}

	for _, name := range []string{"page.pdf", "page.json", "zones.json", "content.json"} {
		if info, err := os.Stat(filepath.Join(pageDir, name)); err != nil || info.Size() == 0 {
			t.Fatalf("%s missing or empty: %v", name, err)
		}
	}

	zones := mustRead(t, filepath.Join(pageDir, "zones.json"))
	if err := contracts.Validate(contracts.Zones, zones); err != nil {
		t.Fatal(err)
	}

	compareGolden(t, filepath.Join("testdata", "decision.zones.json"), canonical(t, zones))

	pageJSON := mustRead(t, filepath.Join(pageDir, "page.json"))
	if err := contracts.Validate(contracts.Page, pageJSON); err != nil {
		t.Fatal(err)
	}

	assertZoneSet(t, zones, "mark-tl", "mark-tr", "mark-bl", "mark-br", "qr", "choice-A", "choice-B", "otherwise", "sketch")
}

func TestRenderDecision_GoldenImage(t *testing.T) {
	t.Parallel()

	tp := typst(t)
	device, _ := domain.Lookup(domain.DefaultDevice)
	h := renderpage.New(nil, tp, t.TempDir(), device, func() time.Time { return fixedNow })
	pageDir := filepath.Join(t.TempDir(), "page")

	if _, err := h.Render(context.Background(), decisionQuestion(t), questionID, 1, pageID, pageDir); err != nil {
		t.Fatal(err)
	}

	// The same inputs rendered to PNG at a low resolution, compared pixel by pixel.
	pngPath := filepath.Join(pageDir, "page.png")
	if err := h.RenderPNG(context.Background(), decisionQuestion(t), questionID, 1, pageID, pngPath, 48); err != nil {
		t.Fatal(err)
	}

	got := decodePNG(t, mustRead(t, pngPath))
	golden := filepath.Join("testdata", "decision.png")

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, mustRead(t, pngPath), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	want := decodePNG(t, mustRead(t, golden))
	if diff := pixelDiff(got, want); diff > 0.002 {
		t.Fatalf("rendered page differs from the golden in %.2f%% of pixels (UPDATE_GOLDEN=1 to accept)", diff*100)
	}
}

func TestQR_PayloadIsAlphanumeric(t *testing.T) {
	t.Parallel()

	device, _ := domain.Lookup(domain.DefaultDevice)

	page, _, err := domain.FromQuestion(decisionQuestion(t), pageID, questionID, 1, device, fixedNow)
	if err != nil {
		t.Fatal(err)
	}

	if page.QR.Payload != "INK:"+pageID+":1" {
		t.Fatalf("payload = %q", page.QR.Payload)
	}

	for _, r := range page.QR.Payload {
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') && r != ':' {
			t.Fatalf("payload %q is not QR alphanumeric (%q)", page.QR.Payload, r)
		}
	}
}

func TestDevices_SizesAndLookup(t *testing.T) {
	t.Parallel()

	d, err := domain.Lookup("supernote-a5x")
	if err != nil {
		t.Fatal(err)
	}

	w, h := d.SizeMM()
	if w != 157.79 || h != 210.39 {
		t.Fatalf("a5x size = %v x %v mm", w, h)
	}

	if _, err := domain.Lookup("kindle"); !errors.Is(err, domain.ErrUnknownDevice) {
		t.Fatalf("err = %v", err)
	}

	if len(domain.Devices()) < 5 {
		t.Fatalf("devices = %v", domain.Devices())
	}
}

// --- helpers ---

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	b, err := os.ReadFile(path) //nolint:gosec // test files
	if err != nil {
		t.Fatal(err)
	}

	return b
}

// canonical re-indents JSON so the golden is readable and stable.
func canonical(t *testing.T, raw []byte) []byte {
	t.Helper()

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}

	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	return append(out, '\n')
}

func compareGolden(t *testing.T, golden string, got []byte) {
	t.Helper()

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(golden) //nolint:gosec // test files
	if err != nil {
		t.Fatalf("no golden %s (UPDATE_GOLDEN=1 to create): %v", golden, err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("%s differs from the golden (UPDATE_GOLDEN=1 to accept):\n%s", golden, got)
	}
}

func assertZoneSet(t *testing.T, zonesDoc []byte, want ...string) {
	t.Helper()

	var doc struct {
		Zones []struct {
			Zone string  `json:"zone"`
			W    float64 `json:"w"`
			H    float64 `json:"h"`
			X    float64 `json:"x"`
			Y    float64 `json:"y"`
		} `json:"zones"`
	}

	if err := json.Unmarshal(zonesDoc, &doc); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	for _, z := range doc.Zones {
		seen[z.Zone] = true

		if z.W <= 0 || z.H <= 0 || z.X < 0 || z.Y < 0 {
			t.Errorf("zone %s has a bad box: %+v", z.Zone, z)
		}
	}

	for _, w := range want {
		if !seen[w] {
			t.Errorf("zone %s missing; got %v", w, seen)
		}
	}
}

func decodePNG(t *testing.T, raw []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}

	return img
}

// pixelDiff is the share of pixels whose luminance differs noticeably.
func pixelDiff(a, b image.Image) float64 {
	if a.Bounds() != b.Bounds() {
		return 1
	}

	var diff, total int

	bounds := a.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			total++

			ra, ga, ba, _ := a.At(x, y).RGBA()
			rb, gb, bb, _ := b.At(x, y).RGBA()

			la := int(ra+ga+ba) / 3 >> 8
			lb := int(rb+gb+bb) / 3 >> 8

			if la-lb > 24 || lb-la > 24 {
				diff++
			}
		}
	}

	return float64(diff) / float64(total)
}
