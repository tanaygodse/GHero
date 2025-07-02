package main

import (
	"fmt"
	"math"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

// AudioManager handles all audio playback for the game
type AudioManager struct {
	sampleRate    beep.SampleRate
	isInitialized bool
	isPlaying     bool
	volume        float64
	testMode      bool // For testing with simple tones
	
	// Audio synthesis
	musicStream   *MIDIAudioStreamer
	currentTime   float64
	startTime     time.Time
}

// MIDIAudioStreamer generates audio from MIDI notes
type MIDIAudioStreamer struct {
	notes        []MIDINote
	sampleRate   beep.SampleRate
	currentSample int64
	startTime    time.Time
}

// NewAudioManager creates a new audio manager
func NewAudioManager() *AudioManager {
	return &AudioManager{
		sampleRate: beep.SampleRate(44100),
		volume:     1.0,
	}
}

// Initialize sets up the audio system
func (am *AudioManager) Initialize() error {
	// Initialize beep speaker with small buffer for low latency
	err := speaker.Init(am.sampleRate, am.sampleRate.N(time.Second/20))
	if err != nil {
		return fmt.Errorf("failed to initialize speaker: %v", err)
	}
	
	am.isInitialized = true
	fmt.Println("Audio system initialized successfully")
	return nil
}

// LoadMIDITrack prepares audio from MIDI notes
func (am *AudioManager) LoadMIDITrack(notes []MIDINote) error {
	if !am.isInitialized {
		return fmt.Errorf("audio manager not initialized")
	}
	
	am.musicStream = &MIDIAudioStreamer{
		notes:      notes,
		sampleRate: am.sampleRate,
	}
	
	fmt.Printf("Loaded MIDI track with %d notes for audio playback\n", len(notes))
	return nil
}

// StartPlayback begins audio playback
func (am *AudioManager) StartPlayback() error {
	fmt.Printf("DEBUG: StartPlayback called - initialized=%v, musicStream=%v\n", 
		am.isInitialized, am.musicStream != nil)
	
	if !am.isInitialized || am.musicStream == nil {
		return fmt.Errorf("audio not ready for playback")
	}
	
	if am.isPlaying {
		fmt.Println("DEBUG: Audio already playing")
		return nil // Already playing
	}
	
	am.startTime = time.Now()
	am.musicStream.startTime = am.startTime
	am.musicStream.currentSample = 0
	
	fmt.Printf("DEBUG: Audio startTime set to %v, notes count: %d\n", 
		am.startTime, len(am.musicStream.notes))
	
	// Create a volume-controlled streamer
	volumeStreamer := &beep.Ctrl{Streamer: am.musicStream, Paused: false}
	volume := &effects.Volume{
		Streamer: volumeStreamer,
		Base:     2,
		Volume:   0, // Use 0 dB (full volume) for testing
		Silent:   false,
	}
	
	speaker.Play(volume)
	am.isPlaying = true
	
	fmt.Printf("DEBUG: Audio playback started successfully - volume=%.1f\n", am.volume)
	return nil
}

// StopPlayback stops audio playback
func (am *AudioManager) StopPlayback() {
	if am.isPlaying {
		speaker.Clear()
		am.isPlaying = false
		fmt.Println("Audio playback stopped")
	}
}

// Update updates the audio manager state
func (am *AudioManager) Update() {
	if am.isPlaying && !am.startTime.IsZero() {
		am.currentTime = time.Since(am.startTime).Seconds()
	}
}

// GetCurrentTime returns the current playback time
func (am *AudioManager) GetCurrentTime() float64 {
	return am.currentTime
}

// SetVolume sets the playback volume (0.0 to 1.0)
func (am *AudioManager) SetVolume(volume float64) {
	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}
	am.volume = volume
}

// IsPlaying returns whether audio is currently playing
func (am *AudioManager) IsPlaying() bool {
	return am.isPlaying
}

// Cleanup releases audio resources
func (am *AudioManager) Cleanup() {
	am.StopPlayback()
	// Beep speaker cleanup is automatic
}

// Stream implements beep.Streamer for MIDI audio generation
func (ms *MIDIAudioStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if ms.startTime.IsZero() {
		fmt.Printf("DEBUG: Audio stream called but startTime is zero\n")
		return 0, false
	}
	
	currentTime := time.Since(ms.startTime).Seconds()
	
	// Debug: Print timing info less frequently
	if ms.currentSample%(int64(ms.sampleRate)*5) == 0 { // Every 5 seconds
		activeNotes := 0
		for _, note := range ms.notes {
			if currentTime >= note.StartTime && currentTime < note.StartTime+note.Duration {
				activeNotes++
			}
		}
		fmt.Printf("DEBUG: Audio time=%.1fs, MIDI notes active=%d\n", 
			currentTime, activeNotes)
	}
	
	for i := range samples {
		// Calculate the time for this sample
		sampleTime := currentTime + float64(i)/float64(ms.sampleRate)
		
		// Generate audio by synthesizing active MIDI notes
		left, right := ms.synthesizeAtTime(sampleTime)
		
		samples[i][0] = left
		samples[i][1] = right
		
		ms.currentSample++
	}
	
	return len(samples), true
}

// Err implements beep.Streamer
func (ms *MIDIAudioStreamer) Err() error {
	return nil
}

// synthesizeAtTime generates audio samples for a specific time
func (ms *MIDIAudioStreamer) synthesizeAtTime(currentTime float64) (float64, float64) {
	var left, right float64
	var activeNoteCount int
	
	// Simple test tone (440Hz for first 2 seconds) to verify audio works
	if currentTime >= 0 && currentTime < 2.0 {
		testFreq := 440.0 // A4 note
		testPhase := 2 * math.Pi * testFreq * currentTime
		testSample := 0.2 * math.Sin(testPhase)
		left += testSample
		right += testSample
		activeNoteCount = 1
	}
	
	// Simple synthesis: find active notes and generate sine waves
	for _, note := range ms.notes {
		noteStart := note.StartTime
		noteEnd := note.StartTime + note.Duration
		
		// Check if this note should be playing at the current time
		if currentTime >= noteStart && currentTime < noteEnd {
			activeNoteCount++
			
			// Convert MIDI pitch to frequency
			frequency := midiToFrequency(note.Pitch)
			
			// Generate sine wave
			phase := 2 * math.Pi * frequency * currentTime
			amplitude := 0.2 // Reduced to avoid overload with test tone
			
			// Simple envelope (fade in/out to avoid clicks)
			envelope := 1.0
			fadeTime := 0.05 // 50ms fade
			
			if currentTime-noteStart < fadeTime {
				envelope = (currentTime - noteStart) / fadeTime
			} else if noteEnd-currentTime < fadeTime {
				envelope = (noteEnd - currentTime) / fadeTime
			}
			
			sample := amplitude * envelope * math.Sin(phase)
			
			// Reduce volume per note when multiple notes are playing
			if activeNoteCount > 1 {
				sample = sample / float64(activeNoteCount)
			}
			
			// Pan based on lane (left, center, right)
			switch note.Lane {
			case 0: // Left lane - more left channel
				left += sample * 0.8
				right += sample * 0.4
			case 1: // Middle lane - center
				left += sample * 0.6
				right += sample * 0.6
			case 2: // Right lane - more right channel
				left += sample * 0.4
				right += sample * 0.8
			default:
				left += sample * 0.5
				right += sample * 0.5
			}
		}
	}
	
	// Debug: Print sample info less frequently
	if activeNoteCount > 0 && ms.currentSample%44100 == 0 { // Every second when notes are active
		fmt.Printf("DEBUG: Audio active - notes: %d, samples: %.3f/%.3f\n", 
			activeNoteCount, left, right)
	}
	
	// Clamp values to prevent distortion
	if left > 1.0 {
		left = 1.0
	} else if left < -1.0 {
		left = -1.0
	}
	
	if right > 1.0 {
		right = 1.0
	} else if right < -1.0 {
		right = -1.0
	}
	
	return left, right
}

// midiToFrequency converts a MIDI note number to frequency in Hz
func midiToFrequency(midiNote int) float64 {
	// A4 (MIDI note 69) = 440 Hz
	return 440.0 * math.Pow(2.0, float64(midiNote-69)/12.0)
}