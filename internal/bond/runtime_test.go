package bond

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAfterWriteRejectsFeedRollback(t *testing.T) {
	min := ""
	get := func() (string, error) { return min, nil }
	set := func(v string) error { min = v; return nil }
	if err := AfterWrite(get, set, "section=up feed=12"); err != nil {
		t.Fatal(err)
	}
	if err := AfterWrite(get, set, "section=dn feed=4"); err == nil {
		t.Fatal("expected feed anti-rollback to reject a lower feed")
	}
}

func TestGrowBondCodesNoWriteThrough(t *testing.T) {
	dst := make([]byte, 2, 8)
	copy(dst, []byte("AB"))
	got := GrowBondCodes(dst, 'C')
	got[0] = 'X'
	if dst[0] != 'A' {
		t.Fatal("GrowBondCodes wrote through into the bond code buffer")
	}
}

func TestBondBagSetGet(t *testing.T) {
	bag := NewBondBag()
	bag.Set("up", 12)
	if bag.Get("up") != 12 {
		t.Fatal("feed not stored")
	}
}

func TestExportBondFileRejectsEscape(t *testing.T) {
	if _, err := ExportBondFile(t.TempDir(), filepath.Join("..", "boot")); err == nil {
		t.Fatal("expected path escape to be rejected")
	}
}

func TestWrapBondDeniedIs(t *testing.T) {
	err := WrapBondDenied("occupy", "up")
	if !errors.Is(err, ErrBond) {
		t.Fatalf("lost ErrBond: %v", err)
	}
}

func TestCopyFeedHeadIndependent(t *testing.T) {
	src := []byte{0x7f, 'E', 'L', 'F', 1, 2, 3, 4}
	got := CopyFeedHead(src, 4)
	got[0] = 0
	if src[0] != 0x7f {
		t.Fatal("CopyFeedHead aliased the feed blob")
	}
}

func TestWaitOccupyHonorsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	err := WaitOccupy(ctx, 600*time.Millisecond)
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if time.Since(start) > 250*time.Millisecond {
		t.Fatalf("WaitOccupy ignored cancel, elapsed=%s", time.Since(start))
	}
}

func TestWaitOccupyPreCancelledReturnsImmediately(t *testing.T) {
	// Regression: a context already cancelled before the call must not sleep
	// the full duration before releasing the bypass wait.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := WaitOccupy(ctx, 600*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("WaitOccupy slept through pre-cancel, elapsed=%s", elapsed)
	}
}

func TestWaitOccupyCompletesWithoutCancel(t *testing.T) {
	// Regression: the normal path (no cancel) still returns nil after the wait.
	ctx := context.Background()
	start := time.Now()
	err := WaitOccupy(ctx, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil on normal completion, got %v", err)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Fatalf("WaitOccupy returned too early, elapsed=%s", elapsed)
	}
}

func TestDumpOccupancyPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "occupancy.txt")
	body := "section=up feed=12\n"
	if err := DumpOccupancy(path, body); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != body {
		t.Fatalf("got %q", b)
	}
}

func TestNilSectionLineName(t *testing.T) {
	var sec *Section
	if sec.LineName() != "" {
		t.Fatalf("got %q", sec.LineName())
	}
}

func TestParseOccupancyJSONRejectsInvalid(t *testing.T) {
	if _, err := ParseOccupancyJSON([]byte("feed=12")); err == nil {
		t.Fatal("expected JSON error")
	}
}