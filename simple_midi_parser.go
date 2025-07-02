package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// SimpleMIDIParser provides basic MIDI parsing functionality
type SimpleMIDIParser struct {
	data         []byte
	position     int
	ticksPerBeat int
	tempo        int // microseconds per beat
}

// NewSimpleMIDIParser creates a new simple MIDI parser
func NewSimpleMIDIParser() *SimpleMIDIParser {
	return &SimpleMIDIParser{
		tempo: 500000, // Default 120 BPM (500000 microseconds per beat)
	}
}

// ParseFile parses a MIDI file and extracts note events
func (p *SimpleMIDIParser) ParseFile(filepath string) ([]MIDINote, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	
	p.data = data
	p.position = 0
	
	// Parse header
	if err := p.parseHeader(); err != nil {
		return nil, fmt.Errorf("failed to parse header: %v", err)
	}
	
	// Parse tracks and extract notes
	notes := make([]MIDINote, 0)
	for p.position < len(p.data) {
		trackNotes, err := p.parseTrack()
		if err != nil {
			fmt.Printf("Warning: failed to parse track: %v\n", err)
			break
		}
		notes = append(notes, trackNotes...)
	}
	
	return notes, nil
}

// parseHeader parses the MIDI file header
func (p *SimpleMIDIParser) parseHeader() error {
	if len(p.data) < 14 {
		return fmt.Errorf("file too short for header")
	}
	
	// Check MThd signature
	if string(p.data[0:4]) != "MThd" {
		return fmt.Errorf("invalid MIDI header signature")
	}
	
	// Skip header length (should be 6)
	p.position = 8
	
	// Read format, tracks, and division
	format := binary.BigEndian.Uint16(p.data[p.position:])
	numTracks := binary.BigEndian.Uint16(p.data[p.position+2:])
	division := binary.BigEndian.Uint16(p.data[p.position+4:])
	
	p.position += 6
	p.ticksPerBeat = int(division)
	
	fmt.Printf("MIDI Header: Format %d, %d tracks, %d ticks per beat\n", 
		format, numTracks, p.ticksPerBeat)
	
	return nil
}

// parseTrack parses a single MIDI track
func (p *SimpleMIDIParser) parseTrack() ([]MIDINote, error) {
	if p.position+8 > len(p.data) {
		return nil, fmt.Errorf("not enough data for track header")
	}
	
	// Check MTrk signature
	if string(p.data[p.position:p.position+4]) != "MTrk" {
		return nil, fmt.Errorf("invalid track header signature")
	}
	
	// Read track length
	trackLength := binary.BigEndian.Uint32(p.data[p.position+4:])
	trackStart := p.position + 8
	trackEnd := trackStart + int(trackLength)
	
	fmt.Printf("Parsing track: %d bytes\n", trackLength)
	
	p.position = trackStart
	
	notes := make([]MIDINote, 0)
	activeNotes := make(map[int]*MIDINote) // pitch -> note
	
	currentTick := 0
	runningStatus := byte(0)
	
	for p.position < trackEnd {
		// Read delta time
		deltaTime, err := p.readVariableLength()
		if err != nil {
			break
		}
		currentTick += deltaTime
		
		if p.position >= trackEnd {
			break
		}
		
		// Read event
		eventByte := p.data[p.position]
		p.position++
		
		var status byte
		if eventByte >= 0x80 {
			// Status byte
			status = eventByte
			runningStatus = status
		} else {
			// Data byte, use running status
			status = runningStatus
			p.position-- // Back up to re-read as data
		}
		
		// Handle different event types
		switch status & 0xF0 {
		case 0x90: // Note On
			if p.position+2 > trackEnd {
				break
			}
			pitch := int(p.data[p.position])
			velocity := int(p.data[p.position+1])
			p.position += 2
			
			if velocity > 0 {
				// Start new note
				note := &MIDINote{
					Pitch:     pitch,
					Velocity:  velocity,
					StartTime: p.ticksToSeconds(currentTick),
					Duration:  0,
					Lane:      0, // Will be assigned later
				}
				activeNotes[pitch] = note
			} else {
				// Note on with velocity 0 = note off
				if activeNote, exists := activeNotes[pitch]; exists {
					activeNote.Duration = p.ticksToSeconds(currentTick) - activeNote.StartTime
					notes = append(notes, *activeNote)
					delete(activeNotes, pitch)
				}
			}
			
		case 0x80: // Note Off
			if p.position+2 > trackEnd {
				break
			}
			pitch := int(p.data[p.position])
			p.position += 2 // Skip velocity
			
			if activeNote, exists := activeNotes[pitch]; exists {
				activeNote.Duration = p.ticksToSeconds(currentTick) - activeNote.StartTime
				notes = append(notes, *activeNote)
				delete(activeNotes, pitch)
			}
			
		case 0xFF: // Meta event
			if p.position >= trackEnd {
				break
			}
			metaType := p.data[p.position]
			p.position++
			
			length, err := p.readVariableLength()
			if err != nil || p.position+length > trackEnd {
				break
			}
			
			// Handle tempo changes
			if metaType == 0x51 && length == 3 {
				tempo := int(p.data[p.position])<<16 | 
						int(p.data[p.position+1])<<8 | 
						int(p.data[p.position+2])
				p.tempo = tempo
				fmt.Printf("Tempo change: %d microseconds per beat\n", tempo)
			}
			
			p.position += length
			
		default:
			// Skip other events
			if status < 0xF0 {
				// Channel message, skip appropriate number of data bytes
				dataBytes := []int{2, 2, 2, 2, 1, 1, 2}
				msgType := int((status & 0x70) >> 4)
				if msgType < len(dataBytes) {
					p.position += dataBytes[msgType]
				}
			}
		}
		
		if p.position >= len(p.data) {
			break
		}
	}
	
	// Add any remaining active notes
	for _, activeNote := range activeNotes {
		activeNote.Duration = 0.5 // Default duration
		notes = append(notes, *activeNote)
	}
	
	p.position = trackEnd
	
	fmt.Printf("Extracted %d notes from track\n", len(notes))
	return notes, nil
}

// readVariableLength reads a MIDI variable-length quantity
func (p *SimpleMIDIParser) readVariableLength() (int, error) {
	value := 0
	for i := 0; i < 4; i++ {
		if p.position >= len(p.data) {
			return 0, fmt.Errorf("unexpected end of data")
		}
		
		b := p.data[p.position]
		p.position++
		
		value = (value << 7) | int(b&0x7F)
		
		if (b & 0x80) == 0 {
			break
		}
	}
	return value, nil
}

// ticksToSeconds converts MIDI ticks to seconds
func (p *SimpleMIDIParser) ticksToSeconds(ticks int) float64 {
	// Convert ticks to seconds using current tempo
	secondsPerTick := float64(p.tempo) / (float64(p.ticksPerBeat) * 1000000.0)
	return float64(ticks) * secondsPerTick
}