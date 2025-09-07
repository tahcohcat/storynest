//go:build unix

package tts

import "syscall"

// pauseProcess pauses the eSpeak process on Unix systems
func (e *ESpeakEngine) pauseProcess() error {
	return e.cmd.Process.Signal(syscall.SIGSTOP)
}

// resumeProcess resumes the eSpeak process on Unix systems
func (e *ESpeakEngine) resumeProcess() error {
	return e.cmd.Process.Signal(syscall.SIGCONT)
}
