package volumelock

import (
	"math"
	"sync"
	"time"
)

type Device interface {
	Level() (float64, error)
	SetLevel(level float64) error
}

type Snapshot struct {
	Level     float64
	Target    float64
	Locked    bool
	Available bool
	Err       error
}

type Options struct {
	Interval  time.Duration
	Tolerance float64
	// OnChange runs while serial is held, so it must not call methods that change state.
	OnChange func(Snapshot)
}

const (
	defaultInterval  = 250 * time.Millisecond
	defaultTolerance = 0.005
)

type Controller struct {
	dev  Device
	opts Options

	// serial orders state changes and OnChange calls; take it before mu.
	serial    sync.Mutex
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

func New(dev Device, opts Options) *Controller {
	if opts.Interval <= 0 {
		opts.Interval = defaultInterval
	}
	if opts.Tolerance <= 0 {
		opts.Tolerance = defaultTolerance
	}
	return &Controller{dev: dev, opts: opts}
}

func (c *Controller) Restore(target float64, locked bool) {
	c.serial.Lock()
	defer c.serial.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.target = clamp(target)
	c.locked = locked
}

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

func (c *Controller) SetLevel(level float64) error {
	return c.do(func() error {
		level = clamp(level)
		c.target = level
		return c.write(level)
	})
}

func (c *Controller) Lock() error {
	return c.do(c.lockLocked)
}

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

func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotLocked()
}

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
