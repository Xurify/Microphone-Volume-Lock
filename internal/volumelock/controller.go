// Package volumelock keeps one capture device at a chosen level.
//
// The Controller holds the policy: the target level, whether the lock is on,
// and when the device has to be rewritten. Platform code only implements Device.
package volumelock

import (
	"math"
	"sync"
	"time"
)

// Device reads and writes the master level of one capture endpoint.
// Both values use a 0–1 scale.
type Device interface {
	Level() (float64, error)
	SetLevel(level float64) error
}

type Snapshot struct {
	Level     float64 // level the device reported on the last poll
	Target    float64 // level the slider holds; follows the device while unlocked
	Locked    bool
	Available bool  // the device answered the last poll
	Err       error // why the device is unavailable; nil otherwise
}

// Options configures a Controller. Zero values select the defaults.
type Options struct {
	// Interval is the poll period. Default 250ms.
	Interval time.Duration
	// Tolerance is how far the device may drift from the target before it is
	// rewritten. Default 0.005 (half a percent).
	Tolerance float64
	// OnChange is called when a visible part of the Snapshot changes. It runs
	// without the controller's data lock held, but it must not call the methods
	// that change state (Lock, Unlock, Toggle, SetLevel, Restore, Run, Close).
	OnChange func(Snapshot)
}

const (
	defaultInterval  = 250 * time.Millisecond
	defaultTolerance = 0.005
)

type Controller struct {
	dev  Device
	opts Options

	// serial orders all state changes and OnChange calls. It is taken before mu.
	serial sync.Mutex
	// mu guards the fields below. It is only held briefly.
	mu        sync.Mutex
	level     float64
	target    float64
	locked    bool
	available bool
	err       error

	lastKey snapshotKey
	hasKey  bool

	loopMu sync.Mutex
	stop   chan struct{}
	done   chan struct{}
}

// New returns a Controller for dev. It does not touch the device until Run.
func New(dev Device, opts Options) *Controller {
	if opts.Interval <= 0 {
		opts.Interval = defaultInterval
	}
	if opts.Tolerance <= 0 {
		opts.Tolerance = defaultTolerance
	}
	return &Controller{dev: dev, opts: opts}
}

// Restore sets the saved state. Call it before Run. A locked restore is kept
// even when the device is missing; the poll loop applies it once the device
// appears.
func (c *Controller) Restore(target float64, locked bool) {
	c.serial.Lock()
	defer c.serial.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.target = clamp(target)
	c.locked = locked
}

// Run polls the device now and then every Interval. Calling Run while the loop
// is already running does nothing.
func (c *Controller) Run() {
	c.loopMu.Lock()
	defer c.loopMu.Unlock()
	if c.stop != nil {
		return
	}
	c.stop = make(chan struct{})
	c.done = make(chan struct{})
	c.do(c.poll)
	go c.loop(c.stop, c.done)
}

// Close stops the poll loop and waits for it to exit. The lock state does not
// change, so the next Run enforces it again.
func (c *Controller) Close() {
	c.loopMu.Lock()
	stop, done := c.stop, c.done
	c.stop, c.done = nil, nil
	c.loopMu.Unlock()
	if stop == nil {
		return
	}
	close(stop)
	<-done
}

// SetLevel moves the device to level (0–1) right away. While locked, the new
// level also becomes the level the lock holds.
func (c *Controller) SetLevel(level float64) error {
	return c.do(func() error {
		level = clamp(level)
		c.target = level
		return c.write(level)
	})
}

// Lock holds the current target level. If the device cannot be written, the
// lock stays off and the error is returned.
func (c *Controller) Lock() error {
	return c.do(c.lockLocked)
}

// Unlock releases the lock. The device keeps its current level.
func (c *Controller) Unlock() error {
	return c.do(func() error {
		c.locked = false
		return nil
	})
}

func (c *Controller) Toggle() error {
	return c.do(func() error {
		if c.locked {
			c.locked = false
			return nil
		}
		return c.lockLocked()
	})
}

// Snapshot returns the current state. It never touches the device.
func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotLocked()
}

// do runs fn under the controller's locks and reports the new snapshot if it
// differs from the last one reported.
func (c *Controller) do(fn func() error) error {
	c.serial.Lock()
	defer c.serial.Unlock()

	c.mu.Lock()
	err := fn()
	snap := c.snapshotLocked()
	key := keyOf(snap)
	changed := !c.hasKey || key != c.lastKey
	c.hasKey, c.lastKey = true, key
	c.mu.Unlock()

	if changed && c.opts.OnChange != nil {
		c.opts.OnChange(snap)
	}
	return err
}

func (c *Controller) loop(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(c.opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c.do(c.poll)
		}
	}
}

// poll reads the device. While unlocked, the target follows the device. While
// locked, the device is rewritten whenever it drifts past the tolerance.
// Caller holds mu.
func (c *Controller) poll() error {
	level, err := c.dev.Level()
	if err != nil {
		c.available, c.err = false, err
		return err
	}
	c.level, c.available, c.err = level, true, nil

	if !c.locked {
		c.target = level
		return nil
	}
	if math.Abs(level-c.target) > c.opts.Tolerance {
		return c.write(c.target)
	}
	return nil
}

// Caller holds mu.
func (c *Controller) write(level float64) error {
	if err := c.dev.SetLevel(level); err != nil {
		c.available, c.err = false, err
		return err
	}
	c.level, c.available, c.err = level, true, nil
	return nil
}

// Caller holds mu.
func (c *Controller) lockLocked() error {
	if c.locked {
		return nil
	}
	if err := c.write(c.target); err != nil {
		return err
	}
	c.locked = true
	return nil
}

func (c *Controller) snapshotLocked() Snapshot {
	return Snapshot{
		Level:     c.level,
		Target:    c.target,
		Locked:    c.locked,
		Available: c.available,
		Err:       c.err,
	}
}

// snapshotKey is the part of a Snapshot that matters to the UI. Levels are
// compared in whole percent so tiny device jitter does not trigger OnChange.
type snapshotKey struct {
	level, target     int
	locked, available bool
	err               string
}

func keyOf(s Snapshot) snapshotKey {
	k := snapshotKey{
		level:     percent(s.Level),
		target:    percent(s.Target),
		locked:    s.Locked,
		available: s.Available,
	}
	if s.Err != nil {
		k.err = s.Err.Error()
	}
	return k
}

func percent(v float64) int {
	return int(math.Round(v * 100))
}

func clamp(v float64) float64 {
	switch {
	case math.IsNaN(v) || v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
