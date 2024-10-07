package main

import (
	"fmt"
	"os/exec"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var isLocked bool
var volumeLevel float64 = 75
var lockButton *widget.Button
var volumeSlider *widget.Slider
var errChan chan error
var nircmdCmd *exec.Cmd

func lockMicrophoneVolume() {
	if isLocked {
		fmt.Println("Unlocked")
		isLocked = false
		lockButton.SetText("Lock Microphone Volume")
		volumeSlider.Enable()
		stopNircmd()
	} else {
		fmt.Println("Locked")
		isLocked = true
		lockButton.SetText("Unlock Microphone Volume")
		volumeSlider.Disable()
		go func() {
			err := setMicrophoneVolume(volumeLevel)
			if err != nil {
				errChan <- err
			}
		}()
	}
}

func setMicrophoneVolume(volume float64) error {
	volumeInt := int(volume / 100 * 65535)
	nircmdCmd = exec.Command("nircmdc.exe", "loop", "172800", "500", "setsysvolume", fmt.Sprintf("%d", volumeInt), "default_record")
	nircmdCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return nircmdCmd.Start()
}

func stopNircmd() {
	if nircmdCmd != nil && nircmdCmd.Process != nil {
		nircmdKillCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", nircmdCmd.Process.Pid))
		nircmdKillCmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
		err := nircmdKillCmd.Start()
		if err != nil {
			fmt.Printf("Error stopping nircmd: %v\n", err)
		}
		nircmdCmd = nil
	}
}

func stopAllNiccmd() {
	cmd := exec.Command("taskkill", "/IM", "nircmdc.exe", "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Error stopping nircmd: %v\n", err)
	}
}

func main() {
	myApp := app.NewWithID("com.xurify.microphone_volume_lock")
	myWindow := myApp.NewWindow("Microphone Volume Lock")
	preferences := myApp.Preferences()

	errChan = make(chan error)

	fmt.Println("Started")

	storedVolumeLevel := preferences.Float("volume")
	if storedVolumeLevel != 0 {
		volumeLevel = storedVolumeLevel
	}

	lockButton = widget.NewButton("Lock Microphone Volume", func() {
		lockMicrophoneVolume()
	})

	data := binding.BindFloat(&volumeLevel)
	volumeLabel := widget.NewLabelWithData(binding.FloatToStringWithFormat(data, "Volume: %.0f%%"))

	volumeSlider = widget.NewSlider(0, 100)
	volumeSlider.SetValue(volumeLevel)
	volumeSlider.OnChanged = func(value float64) {
		if !isLocked {
			volumeLevel = value
			data.Set(value)
			preferences.SetFloat("volume", value)
		}
	}

	stopAllButton := widget.NewButton("Stop All Nircmdc Processes", stopAllNiccmd)
	stopAllButton.Importance = widget.DangerImportance
	stopAllButton.Alignment = widget.ButtonAlignCenter

	content := container.NewVBox(
		lockButton,
		volumeLabel,
		volumeSlider,
		stopAllButton,
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, content.MinSize().Height))
	myWindow.SetCloseIntercept(func() {
		myWindow.Hide()
	})

	// systray.Run(func() {
	// 	systray.SetTitle("Microphone Volume Lock")
	// 	systray.SetTooltip("Microphone Volume Lock")
	// 	systray.AddMenuItem("Show", "Show app").Check()
	// }, func() {})

	menu := fyne.NewMenu("Microphone Volume Lock",
		fyne.NewMenuItem("Show", func() {
			myWindow.Show()
		}),
	)
	if desk, ok := myApp.(desktop.App); ok {
		// res, err := fyne.LoadResourceFromPath("icon.png")
		// if err != nil {
		// 	fmt.Println("Error loading resource: " + err.Error())
		// 	return
		// }
		//desk.SetSystemTrayIcon(res)
		desk.SetSystemTrayMenu(menu)
	}

	go func() {
		for err := range errChan {
			fmt.Printf("Received error from lockVolume: %v\n", err)
			myApp.SendNotification(fyne.NewNotification("Error", err.Error()))
			dialog.ShowError(err, myWindow)
			isLocked = false
			lockButton.SetText("Lock Microphone Volume")
			volumeSlider.Enable()
			stopNircmd()
		}
	}()

	myWindow.ShowAndRun()
}
