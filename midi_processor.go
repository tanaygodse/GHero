package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// MIDIProcessor handles MIDI file parsing and guitar track extraction
type MIDIProcessor struct {
	filePath    string
	tracks      []MIDITrack
	guitarTrack *MIDITrack
}

// MIDITrack represents a single track from a MIDI file
type MIDITrack struct {
	Name        string
	Channel     int
	Instrument  int
	Notes       []MIDINote
	IsGuitar    bool
}

// MIDINote represents a single note event
type MIDINote struct {
	Pitch     int     // MIDI note number (0-127)
	Velocity  int     // Note velocity (0-127)
	StartTime float64 // Time in seconds
	Duration  float64 // Duration in seconds
	Lane      int     // Game lane (0=A, 1=W, 2=D)
}

// NewMIDIProcessor creates a new MIDI processor instance
func NewMIDIProcessor() *MIDIProcessor {
	return &MIDIProcessor{
		tracks: make([]MIDITrack, 0),
	}
}

// LoadMIDI loads and parses a MIDI file
func (mp *MIDIProcessor) LoadMIDI(filePath string) error {
	mp.filePath = filePath
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("MIDI file not found: %s", filePath)
	}
	
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}
	
	mp.filePath = absPath
	fmt.Printf("Loading MIDI file: %s\n", mp.filePath)
	
	// Parse the actual MIDI file
	err = mp.parseMIDIFile()
	if err != nil {
		fmt.Printf("Failed to parse MIDI file, using test data: %v\n", err)
		mp.createTestData()
	}
	
	return nil
}

// parseMIDIFile parses the actual MIDI file using our simple parser
func (mp *MIDIProcessor) parseMIDIFile() error {
	fmt.Printf("Parsing MIDI file: %s\n", mp.filePath)
	
	// Use our simple MIDI parser
	parser := NewSimpleMIDIParser()
	allNotes, err := parser.ParseFile(mp.filePath)
	if err != nil {
		return fmt.Errorf("failed to parse MIDI file: %v", err)
	}
	
	fmt.Printf("Total notes extracted: %d\n", len(allNotes))
	
	// Filter notes to create guitar track
	// For now, we'll use all notes and assume they're guitar notes
	// You can add more sophisticated filtering here based on:
	// - Channel numbers
	// - Pitch ranges
	// - Note patterns
	
	guitarNotes := make([]MIDINote, 0)
	for _, note := range allNotes {
		// Filter out very low or very high notes that don't make sense for guitar
		if note.Pitch >= 40 && note.Pitch <= 84 { // Roughly guitar range
			guitarNotes = append(guitarNotes, note)
		}
	}
	
	fmt.Printf("Guitar notes after filtering: %d\n", len(guitarNotes))
	
	// Create a single guitar track with all the notes
	track := MIDITrack{
		Name:       "Guitar",
		Channel:    0,
		Instrument: 25, // Clean Guitar
		IsGuitar:   true,
		Notes:      guitarNotes,
	}
	
	mp.tracks = []MIDITrack{track}
	
	fmt.Printf("Created guitar track with %d notes\n", len(track.Notes))
	return nil
}

// createTestData creates test data for development
func (mp *MIDIProcessor) createTestData() {
	// Create a mock guitar track for testing
	guitarTrack := MIDITrack{
		Name:       "Guitar",
		Channel:    1,
		Instrument: 25, // Clean Guitar
		IsGuitar:   true,
		Notes: []MIDINote{
			{Pitch: 64, Velocity: 80, StartTime: 1.0, Duration: 0.5, Lane: 0}, // E4 - Lane A
			{Pitch: 67, Velocity: 85, StartTime: 1.5, Duration: 0.5, Lane: 1}, // G4 - Lane W
			{Pitch: 72, Velocity: 90, StartTime: 2.0, Duration: 0.5, Lane: 2}, // C5 - Lane D
			{Pitch: 64, Velocity: 80, StartTime: 2.5, Duration: 1.0, Lane: 0}, // E4 - Lane A (longer note)
			{Pitch: 69, Velocity: 85, StartTime: 3.0, Duration: 0.5, Lane: 1}, // A4 - Lane W
		},
	}
	
	mp.tracks = append(mp.tracks, guitarTrack)
	mp.guitarTrack = &mp.tracks[0]
}

// FindGuitarTrack identifies and returns the guitar track from the MIDI file
func (mp *MIDIProcessor) FindGuitarTrack() (*MIDITrack, error) {
	if len(mp.tracks) == 0 {
		return nil, fmt.Errorf("no tracks loaded")
	}
	
	// Look for guitar track
	for i := range mp.tracks {
		track := &mp.tracks[i]
		if mp.isGuitarTrack(track) {
			track.IsGuitar = true
			mp.guitarTrack = track
			mp.assignLanes(track)
			return track, nil
		}
	}
	
	return nil, fmt.Errorf("no guitar track found")
}

// isGuitarTrack determines if a track contains guitar content
func (mp *MIDIProcessor) isGuitarTrack(track *MIDITrack) bool {
	// Check track name
	if track.Name == "Guitar" || track.Name == "Lead Guitar" || track.Name == "Rhythm Guitar" {
		return true
	}
	
	// Check instrument (guitar instruments are typically 25-32)
	if track.Instrument >= 25 && track.Instrument <= 32 {
		return true
	}
	
	// For now, assume the first track with notes is guitar
	return len(track.Notes) > 0
}

// assignLanes assigns each note to a game lane based on pitch
func (mp *MIDIProcessor) assignLanes(track *MIDITrack) {
	for i := range track.Notes {
		note := &track.Notes[i]
		
		// Map MIDI pitch to lanes
		// Lane 0 (A): Low notes (below 60 - Middle C)
		// Lane 1 (W): Mid notes (60-72)
		// Lane 2 (D): High notes (above 72)
		if note.Pitch < 60 {
			note.Lane = 0 // A key
		} else if note.Pitch <= 72 {
			note.Lane = 1 // W key
		} else {
			note.Lane = 2 // D key
		}
	}
}