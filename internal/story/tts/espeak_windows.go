//go:build windows

package tts

import "fmt"

// pauseProcess attempts to pause the eSpeak process on Windows
// Note: Windows doesn't have SIGSTOP/SIGCONT equivalents, so we simulate pause by killing and restarting
func (e *ESpeakEngine) pauseProcess() error {
	// On Windows, we can't really pause a process the same way
	// We'll just stop it for now - in a real implementation you might
	// want to implement proper pause/resume functionality
	if e.cmd.Process != nil {
		return e.cmd.Process.Kill()
	}
	return fmt.Errorf("no process to pause")
}

// resumeProcess attempts to resume the eSpeak process on Windows
func (e *ESpeakEngine) resumeProcess() error {
	// On Windows, since we can't resume a killed process,
	// we'll just indicate that resume isn't supported
	return fmt.Errorf("resume not supported on Windows - process was terminated")
}
