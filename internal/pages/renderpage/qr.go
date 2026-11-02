package renderpage

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// modules returns the QR code for payload as rows of "0"/"1", without the
// quiet zone (the template leaves white space around it).
func modules(payload string) ([]string, error) {
	q, err := qrcode.New(payload, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("renderpage: qr: %w", err)
	}

	q.DisableBorder = true

	bitmap := q.Bitmap()
	rows := make([]string, 0, len(bitmap))

	for _, row := range bitmap {
		var b strings.Builder

		for _, on := range row {
			if on {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}

		rows = append(rows, b.String())
	}

	return rows, nil
}
