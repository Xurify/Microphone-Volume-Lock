# Microphone-Volume-Lock

Microphone-Volume-Lock is a Windows application designed to lock the microphone volume at a specified level. It provides a graphical user interface for easy control and uses the NirCmd utility to manage system volume settings.

## Features

- Lock microphone volume at a user-defined level
- GUI for easy control of microphone volume
- Ability to stop all NirCmd processes
- Persistent settings across application restarts

## Prerequisites

- Windows operating system
- Go programming language (for development and building)
- NirCmd utility (`nircmdc.exe`) in the same directory as the executable

## Installation

1. Clone this repository or download the source code.
2. Ensure you have Go installed on your system.
3. Place `nircmdc.exe` in the same directory as the source code.

## Usage

### Running the Application

To run the application in development mode:

go run .

### Building the Application

To build the application as a Windows GUI application (without console window):

go build -ldflags -H=windowsgui


This will create an executable named `Microphone-Volume-Lock.exe` in the current directory.

### Using the Application

1. Launch the application.
2. Use the slider to set the desired microphone volume level.
3. Click "Lock Microphone Volume" to lock the volume at the set level.
4. Click "Unlock Microphone Volume" to release the lock.
5. If needed, use the "Stop All Nircmdc Processes" button to terminate any lingering nircmdc processes.

## File Structure

- `main.go`: Main application code
- `start_microphone_volume_lock.bat`: Batch script to start the volume lock (legacy)
- `unlock_microphone_volume.bat`: Batch script to unlock the volume (legacy)
- `hide_cmd_window2.vbs`: VBScript to hide command windows (legacy)
- `lock_microphone_volume.bat`: Batch script to lock the microphone volume (legacy)
- `nircmdc.exe`: NirCmd utility (required, not included in repository)

## Development

The application is written in Go using the Fyne toolkit for the GUI. It saves user preferences in a JSON file and uses NirCmd to control the system volume.

To modify the application:

1. Make changes to `main.go`
2. Test your changes with `go run .`
3. Build the application with `go build -ldflags -H=windowsgui`

## Contributing

Contributions to Microphone-Volume-Lock are welcome. Please feel free to submit pull requests or create issues for bugs and feature requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- NirCmd utility by NirSoft
- Fyne toolkit for Go