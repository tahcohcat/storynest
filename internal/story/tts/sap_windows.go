//go:build windows

package tts

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SAPIEngine implements Windows SAPI TTS
type SAPIEngine struct {
	config  Config
	voice   uintptr
	playing bool
	paused  bool
	mutex   sync.RWMutex
}

// newSAPIEngine creates a new Windows SAPI TTS engine
func newSAPIEngine(config Config) (*SAPIEngine, error) {
	engine := &SAPIEngine{
		config: config,
	}

	return engine, nil
}

func (s *SAPIEngine) Speak(text string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.playing {
		return fmt.Errorf("already playing")
	}

	s.playing = true

	// Simulate async speech
	go func() {
		defer func() {
			s.mutex.Lock()
			s.playing = false
			s.paused = false
			s.mutex.Unlock()
		}()

		// Chunk the text to avoid command line length limits
		chunks := s.chunkText(text, 500) // 500 words per chunk

		for i, chunk := range chunks {
			// Check if we should stop
			s.mutex.RLock()
			shouldStop := !s.playing
			s.mutex.RUnlock()

			if shouldStop {
				break
			}

			// Escape quotes and special characters in the text
			escapedChunk := s.escapeForPowerShell(chunk)

			// Use PowerShell to access Windows Speech API
			cmd := exec.Command("powershell", "-Command",
				fmt.Sprintf(`Add-Type -AssemblyName System.Speech; 
				$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; 
				$synth.Rate = %d; 
				$synth.Volume = %d; 
				$synth.Speak('%s')`,
					int(s.config.Speed*10)-10, // Convert to SAPI range (-10 to 10)
					int(s.config.Volume*100),  // Convert to SAPI range (0 to 100)
					escapedChunk))

			if err := cmd.Run(); err != nil {
				fmt.Printf("SAPI error (chunk %d/%d): %v\n", i+1, len(chunks), err)
				// Continue with next chunk instead of stopping entirely
			}

			// Small pause between chunks to avoid overwhelming the system
			if i < len(chunks)-1 { // Don't pause after the last chunk
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return nil
}

func (s *SAPIEngine) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Kill any running PowerShell processes that might be running SAPI
	exec.Command("taskkill", "/F", "/IM", "powershell.exe", "/T").Run()

	s.playing = false
	s.paused = false
	return nil
}

func (s *SAPIEngine) Pause() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.playing || s.paused {
		return nil
	}

	s.paused = true
	return nil
}

func (s *SAPIEngine) Resume() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.paused {
		return nil
	}

	s.paused = false
	return nil
}

func (s *SAPIEngine) SetVoice(voice string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.config.Voice = voice
	return nil
}

func (s *SAPIEngine) SetSpeed(speed float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if speed <= 0 || speed > 3.0 {
		return fmt.Errorf("speed must be between 0.1 and 3.0")
	}

	s.config.Speed = speed
	return nil
}

func (s *SAPIEngine) SetVolume(volume float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if volume < 0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0 and 1.0")
	}

	s.config.Volume = volume
	return nil
}

func (s *SAPIEngine) IsPlaying() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.playing && !s.paused
}

func (s *SAPIEngine) IsPaused() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.paused
}

func (s *SAPIEngine) GetAvailableVoices() ([]string, error) {
	return []string{"Microsoft David", "Microsoft Zira", "Microsoft Mark"}, nil
}

// chunkText splits text into smaller chunks suitable for SAPI
func (s *SAPIEngine) chunkText(text string, maxWords int) []string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return []string{text}
	}

	var chunks []string
	var currentChunk []string

	for _, word := range words {
		currentChunk = append(currentChunk, word)

		// Check if we should split at a sentence boundary
		if len(currentChunk) >= maxWords {
			// Look for a good breaking point (sentence end)
			chunkText := strings.Join(currentChunk, " ")
			if breakPoint := s.findSentenceBreak(chunkText, len(currentChunk)*3/4); breakPoint > 0 {
				// Split at sentence boundary
				chunk := chunkText[:breakPoint]
				remainder := strings.TrimSpace(chunkText[breakPoint:])

				chunks = append(chunks, chunk)

				// Start next chunk with remainder
				if remainder != "" {
					currentChunk = strings.Fields(remainder)
				} else {
					currentChunk = []string{}
				}
			} else {
				// No good break point, split at word boundary
				chunks = append(chunks, strings.Join(currentChunk, " "))
				currentChunk = []string{}
			}
		}
	}

	// Add remaining words
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, " "))
	}

	return chunks
}

// findSentenceBreak finds the best place to break text at a sentence boundary
func (s *SAPIEngine) findSentenceBreak(text string, minPos int) int {
	// Look for sentence endings after the minimum position
	sentenceEnders := []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}

	bestPos := -1
	for _, ender := range sentenceEnders {
		if pos := strings.Index(text[minPos:], ender); pos != -1 {
			actualPos := minPos + pos + len(ender)
			if bestPos == -1 || actualPos < bestPos {
				bestPos = actualPos
			}
		}
	}

	return bestPos
}

// escapeForPowerShell escapes special characters for PowerShell command execution
func (s *SAPIEngine) escapeForPowerShell(text string) string {
	// Replace single quotes with double single quotes (PowerShell escaping)
	escaped := strings.ReplaceAll(text, "'", "''")

	// Remove or escape other problematic characters
	// Remove control characters that might cause issues
	reg := regexp.MustCompile(`[\x00-\x1f\x7f]`)
	escaped = reg.ReplaceAllString(escaped, " ")

	// Limit length as additional safety measure
	if len(escaped) > 8000 {
		escaped = escaped[:8000] + "..."
	}

	return escaped
}
