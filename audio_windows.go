//go:build windows

package main

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

var (
	errNoMicrophone = errors.New("no recording device is available")
	errDeviceClosed = errors.New("audio device is closed")
)

// COM objects belong to the thread that created them, so all calls run on one locked OS thread.
type captureDevice struct {
	calls chan func(*comState)
	quit  chan struct{}
	done  chan struct{}
	once  sync.Once
}

type comState struct {
	enum     *wca.IMMDeviceEnumerator
	volume   *wca.IAudioEndpointVolume
	volumeID string
}

func newCaptureDevice() (*captureDevice, error) {
	d := &captureDevice{
		calls: make(chan func(*comState)),
		quit:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	ready := make(chan error, 1)
	go d.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return d, nil
}

func (d *captureDevice) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(d.done)

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		ready <- fmt.Errorf("initialise COM: %w", err)
		return
	}
	defer ole.CoUninitialize()

	st := &comState{}
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &st.enum); err != nil {
		ready <- fmt.Errorf("create device enumerator: %w", err)
		return
	}
	ready <- nil

	for {
		select {
		case call := <-d.calls:
			call(st)
		case <-d.quit:
			st.release()
			return
		}
	}
}

func (d *captureDevice) Level() (float64, error) {
	var level float64
	err := d.call(func(st *comState) error {
		vol, err := st.endpoint()
		if err != nil {
			return err
		}
		var f float32
		if err := vol.GetMasterVolumeLevelScalar(&f); err != nil {
			st.dropEndpoint()
			return fmt.Errorf("read volume: %w", err)
		}
		level = float64(f)
		return nil
	})
	return level, err
}

func (d *captureDevice) SetLevel(level float64) error {
	return d.call(func(st *comState) error {
		vol, err := st.endpoint()
		if err != nil {
			return err
		}
		if err := vol.SetMasterVolumeLevelScalar(float32(level), nil); err != nil {
			st.dropEndpoint()
			return fmt.Errorf("set volume: %w", err)
		}
		return nil
	})
}

func (d *captureDevice) Close() error {
	d.once.Do(func() { close(d.quit) })
	<-d.done
	return nil
}

func (d *captureDevice) call(fn func(*comState) error) error {
	result := make(chan error, 1)
	select {
	case d.calls <- func(st *comState) { result <- fn(st) }:
	case <-d.done:
		return errDeviceClosed
	}
	return <-result
}

func (st *comState) endpoint() (*wca.IAudioEndpointVolume, error) {
	var dev *wca.IMMDevice
	if err := st.enum.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &dev); err != nil {
		st.dropEndpoint()
		return nil, errNoMicrophone
	}
	defer dev.Release()

	var id string
	if err := dev.GetId(&id); err != nil {
		return nil, fmt.Errorf("read device id: %w", err)
	}
	if st.volume != nil && id == st.volumeID {
		return st.volume, nil
	}

	st.dropEndpoint()
	var vol *wca.IAudioEndpointVolume
	if err := dev.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &vol); err != nil {
		return nil, fmt.Errorf("open volume control: %w", err)
	}
	st.volume, st.volumeID = vol, id
	return vol, nil
}

func (st *comState) dropEndpoint() {
	if st.volume != nil {
		st.volume.Release()
		st.volume = nil
		st.volumeID = ""
	}
}

func (st *comState) release() {
	st.dropEndpoint()
	if st.enum != nil {
		st.enum.Release()
		st.enum = nil
	}
}
