# Microphone Volume Lock

A Windows application built with Wails that holds your microphone at a fixed volume, so Windows, other apps, or a driver cannot change it during a call or recording.

## Features

- Set the microphone level with the slider; it changes the microphone immediately
- Lock the level so it is held while the app runs
- Lives in the system tray; closing the window keeps the lock active
- Starts hidden with Windows, optionally
- Only one copy of the app runs at a time; starting it again brings the running window forward

No external tools are needed. The app uses the Windows Core Audio API directly.

## Prerequisites

- Windows 10 or later

For development:
- Go 1.23.2 or later (see the `toolchain` line in `go.mod`)
- Node.js and npm
- Wails CLI v2.9.2 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.9.2`)

## Installation

1. Download the latest release from the Releases page.

## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/Xurify/Microphone-Volume-Lock
cd Microphone-Volume-Lock
```

2. Install frontend dependencies:
```bash
cd frontend
npm install
cd ..
```

3. Run in development mode:
```bash
wails dev
```

## Testing

The lock logic and the settings store have unit tests that run on any platform:
```bash
go test -race ./internal/...
```

The Windows audio code runs only on Windows and is checked by compiling it:
```bash
GOOS=windows go vet ./...
```

## Building

```bash
wails build
```

To set the version shown in the About dialog:
```bash
wails build -ldflags "-X main.version=1.1.0"
```

The built executable will be in the `build/bin` directory.

## Usage

### Locking the microphone
1. Launch the application.
2. Move the slider to the level you want. The microphone changes straight away.
3. Click "Lock Microphone Volume". The level is held until you unlock.
4. Click "Unlock Microphone Volume" to release the lock. The microphone keeps its current level.

When the window is closed, the app keeps running in the tray and the lock stays active.

### Startup
Tick "Run at Windows startup" in the window, or use Options > Run at Windows Startup. When started this way the app opens hidden in the tray.

### Keyboard Shortcuts
- `Ctrl+H`: Hide to system tray
- `Ctrl+Q`: Quit application
- `Ctrl+S`: Toggle Windows startup setting

### System Tray
Right-click the tray icon for:
- Show: bring the window back
- Lock / Unlock Volume
- Quit

Hover over the icon to see whether the microphone is locked and at which level.

### Quitting
Quit (tray menu, File > Exit, or `Ctrl+Q`) stops the lock. The next time the app starts, a lock that was on when you quit is turned back on.

### Files
The app stores its files in `%AppData%\MicrophoneVolumeLock`:
- `state.json`: the saved level and lock state
- `app.log`: a log of events and errors (it is restarted once it grows past 1 MB)

If `state.json` is damaged, it is renamed to `state.json.corrupt` and the defaults are used.

## Project Structure

- `main.go`: Application entry point, window configuration, and single-instance setup
- `app.go`: Methods the frontend calls, and the state shared with the UI and the tray
- `menu.go`: Application menu
- `tray.go`: System tray icon and menu
- `settings.go`: Debounced saving of the settings file
- `paths.go`: App data folder and logging
- `audio_windows.go`: Core Audio access for the default recording device
- `startup_windows.go`: Windows startup entry (registry)
- `legacy_windows.go`: Stops copies of the old nircmd helper left by version 1.0
- `internal/volumelock/`: The lock policy (target level, drift correction, device errors)
- `internal/appstate/`: Atomic load and save of the settings file
- `frontend/`: React frontend
  - `src/App.tsx`: Main application interface
  - `src/style.scss`: Application styling
- `build/`: Build artifacts and resources
- `wails.json`: Wails project configuration

## Contributing

Contributions are welcome! Please feel free to submit pull requests or create issues for bugs and feature requests.

## License

This project is licensed under the MIT License. The `LICENSE` file has not been added to this repository yet.

## Acknowledgments

- [Wails](https://wails.io/) - For the Go/React framework
- [go-wca](https://github.com/moutend/go-wca) - Go bindings for Windows Core Audio
- [systray](https://github.com/getlantern/systray) - System tray support
- [React](https://reactjs.org/) - Frontend framework
- [Radix UI](https://www.radix-ui.com/) - UI components
