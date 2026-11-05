package windowfit

type Size struct{ Width, Height int }

type Plan struct {
	Window             Size
	MinWidth, MaxWidth int
}

// Fit sizes a window so its page is contentHeight tall and between minClient
// and defaultClient wide. The frame is the window size minus the viewport.
// The first fit uses the default width; later fits keep the user's width.
func Fit(window, viewport Size, contentHeight, minClient, defaultClient int, first bool) Plan {
	frameW, frameH := window.Width-viewport.Width, window.Height-viewport.Height
	p := Plan{MinWidth: minClient + frameW, MaxWidth: defaultClient + frameW}
	width := window.Width
	if first {
		width = p.MaxWidth
	}
	p.Window = Size{Width: min(max(width, p.MinWidth), p.MaxWidth), Height: contentHeight + frameH}
	return p
}
