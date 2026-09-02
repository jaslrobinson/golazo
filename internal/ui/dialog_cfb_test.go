package ui

import (
	"testing"

	"github.com/jaslrobinson/golazo/internal/api"
)

func TestFieldPositionIndex(t *testing.T) {
	const width = 41 // odd width has an exact midpoint

	if got := fieldPositionIndex(100, width); got != 0 {
		t.Errorf("own goal line (yardsToEndzone=100) should be index 0, got %d", got)
	}
	if got := fieldPositionIndex(0, width); got != width-1 {
		t.Errorf("opponent goal line (yardsToEndzone=0) should be index %d, got %d", width-1, got)
	}
	if got := fieldPositionIndex(50, width); got != (width-1)/2 {
		t.Errorf("midfield (yardsToEndzone=50) should be index %d, got %d", (width-1)/2, got)
	}

	// Out-of-range inputs must clamp, not panic or go out of bounds.
	if got := fieldPositionIndex(-5, width); got != width-1 {
		t.Errorf("negative yardsToEndzone should clamp to opponent goal, got %d", got)
	}
	if got := fieldPositionIndex(150, width); got != 0 {
		t.Errorf("yardsToEndzone > 100 should clamp to own goal, got %d", got)
	}
}

func TestTimeoutDots(t *testing.T) {
	cases := map[int]string{
		3:  "●●●",
		2:  "●●○",
		1:  "●○○",
		0:  "○○○",
		-1: "○○○",
		5:  "●●●",
	}
	for remaining, want := range cases {
		if got := timeoutDots(remaining); got != want {
			t.Errorf("timeoutDots(%d) = %q, want %q", remaining, got, want)
		}
	}
}

func TestOrdinal(t *testing.T) {
	cases := map[int]string{1: "1st", 2: "2nd", 3: "3rd", 4: "4th"}
	for n, want := range cases {
		if got := ordinal(n); got != want {
			t.Errorf("ordinal(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestBucketAverage(t *testing.T) {
	values := []float64{0, 0, 1, 1} // downsample 4 -> 2 buckets
	got := bucketAverage(values, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(got))
	}
	if got[0] != 0 || got[1] != 1 {
		t.Errorf("expected [0 1], got %v", got)
	}

	// Upsampling: fewer values than buckets must still return exactly n buckets.
	up := bucketAverage([]float64{0.5}, 5)
	if len(up) != 5 {
		t.Fatalf("expected 5 buckets, got %d", len(up))
	}
	for i, v := range up {
		if v != 0.5 {
			t.Errorf("bucket %d = %v, want 0.5", i, v)
		}
	}

	if got := bucketAverage(nil, 3); len(got) != 3 {
		t.Errorf("empty input should still return n buckets, got %d", len(got))
	}
}

func TestSparkLevelIndex(t *testing.T) {
	if got := sparkLevelIndex(0); got != 0 {
		t.Errorf("sparkLevelIndex(0) = %d, want 0", got)
	}
	if got := sparkLevelIndex(1); got != len(sparkLevels)-1 {
		t.Errorf("sparkLevelIndex(1) = %d, want %d", got, len(sparkLevels)-1)
	}
	// Out-of-range must clamp into a valid slice index, not panic.
	if got := sparkLevelIndex(-1); got != 0 {
		t.Errorf("sparkLevelIndex(-1) = %d, want 0", got)
	}
	if got := sparkLevelIndex(2); got != len(sparkLevels)-1 {
		t.Errorf("sparkLevelIndex(2) = %d, want %d", got, len(sparkLevels)-1)
	}
}

func TestDescribeBiggestSwing(t *testing.T) {
	points := []api.MomentumPoint{
		{PlayID: "1", HomeWinPct: 0.50},
		{PlayID: "2", HomeWinPct: 0.55},
		{PlayID: "3", HomeWinPct: 0.20}, // biggest swing: away team, -0.35
		{PlayID: "4", HomeWinPct: 0.25},
	}
	got := describeBiggestSwing(points, "Home U", "Away State")
	if got == "" {
		t.Fatal("expected non-empty description")
	}
	if !contains(got, "Away State") {
		t.Errorf("expected swing to favor Away State, got %q", got)
	}

	if got := describeBiggestSwing(points[:1], "H", "A"); got != "" {
		t.Errorf("single point should return empty description, got %q", got)
	}
}

func TestSituationDialog_ZeroValueSituationShowsEmptyState(t *testing.T) {
	// Regression test: a Situation{} with Down==0 (e.g. a provider that only
	// attached a LastPlay fallback) must render the empty state, not a
	// misleading "1st & 0" with the ball on the opponent's goal line.
	dialog := NewSituationDialog("Home", "Away", 10, 7, 1, 2, &api.Situation{LastPlay: "some text"})
	view := dialog.View(120, 40)
	if !contains(view, "No live situation data available") {
		t.Errorf("expected empty-state message for a zero-Down situation, got:\n%s", view)
	}

	// A real situation (Down > 0) should render normally, not the empty state.
	dialog2 := NewSituationDialog("Home", "Away", 10, 7, 1, 2, &api.Situation{
		Down: 2, Distance: 6, YardsToEndzone: 56, DownDistanceText: "2nd & 6",
	})
	view2 := dialog2.View(120, 40)
	if contains(view2, "No live situation data available") {
		t.Errorf("a real situation should not show the empty state, got:\n%s", view2)
	}
}

func TestAxisLine(t *testing.T) {
	for _, width := range []int{20, 45, 82, 200} {
		got := axisLine(width, "Game start", "Now")
		if len(got) != width {
			t.Errorf("axisLine width=%d: got length %d, want %d (line: %q)", width, len(got), width, got)
		}
		if !contains(got, "Now") {
			t.Errorf("axisLine width=%d: right label missing entirely: %q", width, got)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
