# Microphone Volume Lock

A Windows application built with Wails that lets you lock your microphone volume at a specified level, preventing unwanted volume changes during important calls or recordings.

## Features

- Lock microphone volume at any desired level
- System tray integration for quick access
- Built using Wails and React for a native feel

## Prerequisites

- Windows operating system
<!-- - [NirCmd utility](https://www.nirsoft.net/utils/nircmd.html) installed (`nircmdc.exe` needed) -->

For development:
- Go 1.21 or later
- Node.js and npm
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

**Note:** The repository includes `nircmdc.exe` which is embedded into the final executable during build.

## Installation

1. Download the latest release from the Releases page


## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/Xurify/Microphone-Volume-Lock
cd microphone-volume-lock
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

## Building

To build the application:
```bash
wails build
```

The built executable will be in the `build/bin` directory.

## Usage

### Running the Application
1. Launch the application
2. Use the slider to set your desired microphone volume
3. Click "Lock Microphone Volume" to lock at the current level
4. The application will continue running in the system tray when minimized

### Keyboard Shortcuts
- `Ctrl+H`: Hide to system tray
- `Ctrl+Q`: Quit application
- `Ctrl+S`: Toggle Windows startup setting

### System Tray
Right-click the system tray icon for quick access to:
- Show/hide window
- Lock/unlock volume
- Quit application

## Project Structure

- `main.go`: Application entry point and window configuration
- `app.go`: Core application logic and backend functionality
- `frontend/`: React frontend code
  - `src/App.tsx`: Main application interface
  - `src/style.css`: Application styling
- `build/`: Build artifacts and resources
- `wails.json`: Wails project configuration

## Contributing

Contributions are welcome! Please feel free to submit pull requests or create issues for bugs and feature requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Wails](https://wails.io/) - For the Go/React framework
- [NirCmd](https://www.nirsoft.net/utils/nircmd.html) - For Windows volume control
- [React](https://reactjs.org/) - Frontend framework
- [Radix UI](https://www.radix-ui.com/) - UI components