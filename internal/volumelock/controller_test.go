package volumelock

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var errNoMic = errors.New("no microphone")

type fakeDevice struct {
	mu     sync.Mutex
	level  float64
	err    error
	writes []float64
}

func (d *fakeDevice) Level() (float64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.level, d.err
}

func (d *fakeDevice) SetLevel(v float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		return d.err
	}
	d.writes = append(d.writes, v)
	d.level = v
	return nil
}

func (d *fakeDevice) external(level float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.level = level
}

func (d *fakeDevice) fail(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.err = err
}

func (d *fakeDevice) writeCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.writes)
}

func (d *fakeDevice) lastWrite(t *testing.T) float64 {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.writes) == 0 {
		t.Fatal("device was never written")
	}
	return d.writes[len(d.writes)-1]
}

func newTest(dev *fakeDevice, opts Options) *Controller {
	return New(dev, opts)
}

func tick(c *Controller) { c.do(c.pollLocked) }

func approx(a, b float64) bool {
	const eps = 1e-6
	return a-b < eps && b-a < eps
}

func TestUnlockedTargetFollowsDevice(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})

	tick(c)

	s := c.Snapshot()
	if !approx(s.Target, 0.4) || s.Locked || !s.Available {
		t.Fatalf("unlocked snapshot = %+v, want target 0.4, unlocked, available", s)
	}
	if n := dev.writeCount(); n != 0 {
		t.Fatalf("unlocked poll wrote the device %d times", n)
	}
}

func TestSetLevelAppliesImmediatelyWhileUnlocked(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})

	if err := c.SetLevel(0.3); err != nil {
		t.Fatal(err)
	}

	if got := dev.lastWrite(t); !approx(got, 0.3) {
		t.Fatalf("device written with %v, want 0.3", got)
	}
	if s := c.Snapshot(); !approx(s.Target, 0.3) || s.Locked {
		t.Fatalf("snapshot = %+v, want target 0.3 and unlocked", s)
	}
}

func TestLockHoldsLevelAgainstDrift(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})

	if err := c.SetLevel(0.75); err != nil {
		t.Fatal(err)
	}
	if err := c.Lock(); err != nil {
		t.Fatal(err)
	}

	dev.external(0.5)
	tick(c)

	if got := dev.lastWrite(t); !approx(got, 0.75) {
		t.Fatalf("lock did not restore the level: last write %v, want 0.75", got)
	}
	if s := c.Snapshot(); !s.Locked || !approx(s.Level, 0.75) {
		t.Fatalf("snapshot = %+v, want locked at 0.75", s)
	}
}

func TestToleranceAvoidsRewrites(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})
	_ = c.SetLevel(0.75)
	_ = c.Lock()
	before := dev.writeCount()

	dev.external(0.751)
	tick(c)

	if n := dev.writeCount(); n != before {
		t.Fatalf("tiny drift caused %d writes", n-before)
	}
}

func TestUnlockStopsEnforcement(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})
	_ = c.SetLevel(0.75)
	_ = c.Lock()
	if err := c.Unlock(); err != nil {
		t.Fatal(err)
	}
	before := dev.writeCount()

	dev.external(0.2)
	tick(c)

	if n := dev.writeCount(); n != before {
		t.Fatalf("unlocked controller wrote the device %d times", n-before)
	}
	if s := c.Snapshot(); s.Locked || !approx(s.Target, 0.2) {
		t.Fatalf("snapshot = %+v, want unlocked and target following device to 0.2", s)
	}
}

func TestLockFailsWithoutDeviceAndStaysOff(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})
	dev.fail(errNoMic)

	err := c.Lock()

	if !errors.Is(err, errNoMic) {
		t.Fatalf("Lock error = %v, want errNoMic", err)
	}
	if s := c.Snapshot(); s.Locked || s.Available || !errors.Is(s.Err, errNoMic) {
		t.Fatalf("snapshot = %+v, want unlocked, unavailable, errNoMic", s)
	}
}

func TestRestoredLockWaitsForDevice(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})
	c.Restore(0.6, true)
	dev.fail(errNoMic)

	tick(c)
	if s := c.Snapshot(); !s.Locked || s.Available {
		t.Fatalf("snapshot = %+v, want locked but unavailable while device is missing", s)
	}

	dev.fail(nil)
	tick(c)

	if got := dev.lastWrite(t); !approx(got, 0.6) {
		t.Fatalf("restored lock wrote %v, want 0.6", got)
	}
	if s := c.Snapshot(); !s.Locked || !s.Available || s.Err != nil {
		t.Fatalf("snapshot = %+v, want locked and available after device returns", s)
	}
}

func TestDeviceRemovedMidLockKeepsLockAndReports(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})
	_ = c.SetLevel(0.75)
	_ = c.Lock()

	dev.fail(errNoMic)
	tick(c)

	if s := c.Snapshot(); !s.Locked || s.Available || !errors.Is(s.Err, errNoMic) {
		t.Fatalf("snapshot = %+v, want locked, unavailable, errNoMic", s)
	}
}

func TestSetLevelClampsRange(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})

	_ = c.SetLevel(1.7)
	if s := c.Snapshot(); !approx(s.Target, 1) {
		t.Fatalf("target after 1.7 = %v, want 1", s.Target)
	}
	_ = c.SetLevel(-0.2)
	if s := c.Snapshot(); !approx(s.Target, 0) {
		t.Fatalf("target after -0.2 = %v, want 0", s.Target)
	}
}

func TestToggleLocksThenUnlocks(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{})

	if err := c.Toggle(); err != nil {
		t.Fatal(err)
	}
	if !c.Snapshot().Locked {
		t.Fatal("first toggle did not lock")
	}
	if err := c.Toggle(); err != nil {
		t.Fatal(err)
	}
	if c.Snapshot().Locked {
		t.Fatal("second toggle did not unlock")
	}
}

func TestOnChangeOnlyForVisibleChanges(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	var got []Snapshot
	c := newTest(dev, Options{OnChange: func(s Snapshot) { got = append(got, s) }})

	tick(c)
	tick(c)
	dev.external(0.4001)
	tick(c)
	if len(got) != 1 {
		t.Fatalf("notifications after jitter = %d, want 1", len(got))
	}

	dev.external(0.5)
	tick(c)
	if len(got) != 2 {
		t.Fatalf("notifications after a real change = %d, want 2", len(got))
	}
}

func TestRunEnforcesLockAndCloseStopsLoop(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{Interval: time.Millisecond})
	_ = c.SetLevel(0.7)
	_ = c.Lock()

	c.Run()
	defer c.Close()

	dev.external(0.1)
	waitFor(t, func() bool { return dev.lastWrite(t) == 0.7 && dev.writeCount() >= 2 })

	c.Close()
	n := dev.writeCount()
	dev.external(0.2)
	time.Sleep(20 * time.Millisecond)
	if dev.writeCount() != n {
		t.Fatal("loop kept enforcing after Close")
	}
}

func TestConcurrentUseIsSafe(t *testing.T) {
	dev := &fakeDevice{level: 0.4}
	c := newTest(dev, Options{Interval: time.Millisecond})
	c.Run()
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				switch (i + j) % 4 {
				case 0:
					_ = c.Toggle()
				case 1:
					_ = c.SetLevel(float64(j%100) / 100)
				case 2:
					_ = c.Snapshot()
				default:
					_ = c.Lock()
				}
			}
		}(i)
	}
	wg.Wait()
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met in time")
}
