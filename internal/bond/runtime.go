package bond

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrBond = errors.New("bond denied")

func GrowBondCodes(dst []byte, extra byte) []byte {
	out := make([]byte, len(dst)+1)
	copy(out, dst)
	out[len(dst)] = extra
	return out
}

type BondBag struct {
	feeds map[string]int
}

func NewBondBag() *BondBag {
	bag := &BondBag{}
	bag.feeds = make(map[string]int)
	return bag
}

func (b *BondBag) Set(section string, feed int) {
	b.feeds[section] = feed
}

func (b *BondBag) Get(section string) int {
	return b.feeds[section]
}

func ExportBondFile(root, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return "", errors.New("empty bond path")
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("absolute bond path")
	}
	clean := filepath.Clean(rel)
	full := filepath.Join(root, clean)
	relOut, err := filepath.Rel(filepath.Clean(root), full)
	if err != nil {
		return "", err
	}
	if relOut == ".." || strings.HasPrefix(relOut, ".."+string(filepath.Separator)) {
		return "", errors.New("bond path escapes root")
	}
	return full, nil
}

func WrapBondDenied(op, section string) error {
	if strings.TrimSpace(op) == "" {
		op = "occupy"
	}
	if strings.TrimSpace(section) == "" {
		section = "unknown"
	}
	return fmt.Errorf("%s: section %s: %w", op, section, ErrBond)
}

func CopyFeedHead(blob []byte, n int) []byte {
	if n < 0 {
		n = 0
	}
	if n > len(blob) {
		n = len(blob)
	}
	out := make([]byte, n)
	copy(out, blob[:n])
	return out
}

func WaitOccupy(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func DumpOccupancy(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := w.WriteString(body); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

type Section struct {
	Line string
	Feed string
}

func (sec *Section) LineName() string {
	if sec == nil {
		return ""
	}
	return sec.Line
}

func ParseOccupancyJSON(b []byte) (map[string]int, error) {
	var m map[string]int
	if len(b) == 0 {
		return nil, errors.New("empty occupancy")
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}