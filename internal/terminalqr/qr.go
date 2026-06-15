package terminalqr

import (
	"fmt"
	"io"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func Render(w io.Writer, content string, width int) error {
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return err
	}
	bitmap := q.Bitmap()
	if len(bitmap) == 0 {
		return fmt.Errorf("empty QR bitmap")
	}
	quiet := 2
	size := len(bitmap) + quiet*2
	if width > 0 && size > width {
		return fmt.Errorf("terminal width %d is too narrow for QR width %d", width, size)
	}
	white := strings.Repeat("  ", size)
	fmt.Fprintln(w, white)
	for y := 0; y < len(bitmap); y++ {
		fmt.Fprint(w, strings.Repeat("  ", quiet))
		for x := 0; x < len(bitmap[y]); x++ {
			if bitmap[y][x] {
				fmt.Fprint(w, "██")
			} else {
				fmt.Fprint(w, "  ")
			}
		}
		fmt.Fprintln(w, strings.Repeat("  ", quiet))
	}
	fmt.Fprintln(w, white)
	return nil
}

func PNG(content string, size int) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, size)
}
