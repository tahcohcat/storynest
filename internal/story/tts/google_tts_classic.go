package tts

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/texttospeech/apiv1"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

type GoogleClassicTTSEngine struct {
	client    *texttospeech.Client
	ctx       context.Context
	voice     string
	speed     float64
	volume    float64
	isPlaying bool
	ctrl      *beep.Ctrl
	format    beep.Format
	streamer  beep.StreamSeekCloser
	done      chan bool
	mu        sync.Mutex
	cacheDir  string
}

func newGoogleClassicTTSEngine(cacheDir string) (*GoogleClassicTTSEngine, error) {
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS client: %w", err)
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	return &GoogleClassicTTSEngine{
		client:   client,
		ctx:      ctx,
		voice:    "en-GB-Chirp3-HD-Umbriel", //some random default
		speed:    1.0,
		volume:   1.0,
		cacheDir: cacheDir,
	}, nil
}

func (g *GoogleClassicTTSEngine) Speak(text string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	audioCfg := &texttospeechpb.AudioConfig{
		AudioEncoding: texttospeechpb.AudioEncoding_MP3,
	}

	// Chirp voices often don't support speakingRate/pitch/SSML — skip them
	if !strings.Contains(strings.ToLower(g.voice), "chirp") {
		audioCfg.SpeakingRate = g.speed  // supported generally. See docs.
		audioCfg.VolumeGainDb = g.volume // interpret as dB gain (or map to dB as you prefer)
		// audioCfg.Pitch = ...   // set only if voice supports it
	}

	// Cache key
	fileName := filepath.Join(g.cacheDir, fmt.Sprintf("%x", md5Sum(text+g.voice)))

	// todo: increase chunk size
	chunks := splitIntoChunks(text, 4800) // a little under 5000 to be safe
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		// Generate MP3 if not cached
		for chunkIndex, chunk := range chunks {

			// todo: just testing small bits for now
			if chunkIndex > 0 {
				continue
			}

			req := &texttospeechpb.SynthesizeSpeechRequest{
				Input: &texttospeechpb.SynthesisInput{
					InputSource: &texttospeechpb.SynthesisInput_Text{Text: chunk},
				},
				Voice: &texttospeechpb.VoiceSelectionParams{
					LanguageCode: "en-US",
					Name:         "en-US-Chirp3-HD-Charon",
				},
				AudioConfig: &texttospeechpb.AudioConfig{
					AudioEncoding: texttospeechpb.AudioEncoding_MP3,
				},
			}
			resp, err := g.client.SynthesizeSpeech(g.ctx, req)
			if err != nil {
				return fmt.Errorf("failed to synthesize: %w", err)
			}

			chunkName := fmt.Sprintf("%s_%d.mp3", fileName, chunkIndex)
			if err := os.WriteFile(chunkName, resp.AudioContent, 0644); err != nil {
				return fmt.Errorf("failed to write MP3: %w", err)
			}
		}
	}

	for i := 0; i < len(chunks); i++ {
		// Play cached file

		if i > 0 {
			continue
		}

		chunkName := fmt.Sprintf("%s_%d.mp3", fileName, i)
		f, err := os.Open(chunkName)
		if err != nil {
			return fmt.Errorf("failed to open MP3: %w", err)
		}

		streamer, format, err := mp3.Decode(f)
		if err != nil {
			return fmt.Errorf("failed to decode MP3: %w", err)
		}
		g.streamer = streamer
		g.format = format
		g.ctrl = &beep.Ctrl{Streamer: streamer, Paused: false}
		g.done = make(chan bool)

		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		if err != nil {
			return err
		}
		speaker.Play(beep.Seq(g.ctrl, beep.Callback(func() {
			g.isPlaying = false
			g.done <- true
		})))

		g.isPlaying = true

	}
	return nil
}

func (g *GoogleClassicTTSEngine) SetVoice(voice string) error {
	g.voice = voice
	return nil
}

func (g *GoogleClassicTTSEngine) SetSpeed(speed float64) error {
	g.speed = speed
	return nil
}

func (g *GoogleClassicTTSEngine) SetVolume(volume float64) error {
	g.volume = volume
	return nil
}

func (g *GoogleClassicTTSEngine) Stop() error {
	if g.streamer != nil {
		g.streamer.Close()
	}
	g.isPlaying = false
	return nil
}

func (g *GoogleClassicTTSEngine) Pause() error {
	if g.ctrl != nil {
		speaker.Lock()
		g.ctrl.Paused = true
		speaker.Unlock()
	}
	return nil
}

func (g *GoogleClassicTTSEngine) Resume() error {
	if g.ctrl != nil {
		speaker.Lock()
		g.ctrl.Paused = false
		speaker.Unlock()
	}
	return nil
}

func (g *GoogleClassicTTSEngine) IsPlaying() bool {
	return g.isPlaying
}

func (g *GoogleClassicTTSEngine) GetAvailableVoices() ([]string, error) {
	resp, err := g.client.ListVoices(g.ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		return nil, err
	}
	voices := []string{}
	for _, v := range resp.Voices {
		voices = append(voices, v.Name)
	}
	return voices, nil
}

// helper
func md5Sum(s string) string {
	h := md5.New()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func splitIntoChunks(text string, limit int) []string {
	var chunks []string
	runes := []rune(text) // safe for UTF-8
	for i := 0; i < len(runes); i += limit {
		end := i + limit
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}
