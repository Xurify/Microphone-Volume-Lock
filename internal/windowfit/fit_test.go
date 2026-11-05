package windowfit

import "testing"

func TestFirstFitUsesMeasuredFrameAndDefaultWidth(t *testing.T) {
	// The window starts at the 416x266 guess, but its real frame is 16x47.
	got := Fit(Size{416, 266}, Size{400, 266 - 47}, 226, 300, 400, true)
	want := Plan{Window: Size{416, 226 + 47}, MinWidth: 316, MaxWidth: 416}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLaterFitKeepsUserWidthWithinLimits(t *testing.T) {
	frame := Size{16, 40}
	for _, tc := range []struct{ clientWidth, want int }{{350, 350}, {250, 300}, {450, 400}} {
		window := Size{tc.clientWidth + frame.Width, 266}
		viewport := Size{tc.clientWidth, 266 - frame.Height}
		got := Fit(window, viewport, 226, 300, 400, false)
		if got.Window.Width-frame.Width != tc.want {
			t.Errorf("client width %d: got %d, want %d", tc.clientWidth, got.Window.Width-frame.Width, tc.want)
		}
	}
}

func TestHeightFollowsContent(t *testing.T) {
	got := Fit(Size{416, 266}, Size{400, 226}, 240, 300, 400, false)
	if got.Window.Height != 280 {
		t.Fatalf("height = %d, want 280", got.Window.Height)
	}
}
