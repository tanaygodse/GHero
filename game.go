package main

import (
	"fmt"
	"time"
	
	rl "github.com/gen2brain/raylib-go/raylib"
)

// GameState represents the current state of the game
type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
	StateGameOver
)

// Game represents the main game state
type Game struct {
	screenWidth    int32
	screenHeight   int32
	midiProcessor  *MIDIProcessor
	audioManager   *AudioManager
	gameNotes      []GameNote
	score          int32
	combo          int32
	maxCombo       int32
	state          GameState
	gameStartTime  time.Time
	currentTime    float64
	songDuration   float64
	hitLine        float32 // Y position of the hit line
	lanes          [3]Lane
	
	// Statistics
	perfectHits    int32
	goodHits       int32
	okHits         int32
	missedHits     int32
	totalNotes     int32
}

// GameNote represents a note in the game
type GameNote struct {
	StartTime    float64
	Duration     float64
	Lane         int
	Y            float32 // Current Y position on screen
	Width        float32
	Height       float32
	IsActive     bool
	IsHit        bool
	HitAccuracy  HitAccuracy
	
	// Sustained note tracking
	IsPressed       bool    // Whether the key is currently pressed for this note
	PressStartTime  float64 // When the key was first pressed for this note
	IsBeingHeld     bool    // Whether the note is being held correctly
	SustainProgress float64 // How much of the sustain has been completed (0.0 to 1.0)
}

// Lane represents one of the three game lanes
type Lane struct {
	X         float32
	Width     float32
	IsPressed bool
	KeyCode   int32
}

// HitAccuracy represents how accurate a hit was
type HitAccuracy int

const (
	Miss HitAccuracy = iota
	OK
	Good
	Perfect
)

// Game constants
const (
	SCREEN_WIDTH     = 800
	SCREEN_HEIGHT    = 600
	LANE_WIDTH       = 200
	NOTE_HEIGHT      = 40
	HIT_LINE_Y       = 500
	NOTE_SPEED       = 200   // pixels per second
	GAME_DURATION    = 30.0  // Game duration in seconds
	COUNTDOWN_TIME   = 3.0   // Countdown before game starts
)

// NewGame creates a new game instance
func NewGame() *Game {
	// Initialize audio manager
	audioManager := NewAudioManager()
	err := audioManager.Initialize()
	if err != nil {
		fmt.Printf("Warning: Failed to initialize audio: %v\n", err)
		// Continue without audio
	}
	
	game := &Game{
		screenWidth:   SCREEN_WIDTH,
		screenHeight:  SCREEN_HEIGHT,
		audioManager:  audioManager,
		hitLine:      HIT_LINE_Y,
		score:        0,
		combo:        0,
		maxCombo:     0,
		state:        StateMenu,
		songDuration: 0,
		perfectHits:  0,
		goodHits:     0,
		okHits:       0,
		missedHits:   0,
		totalNotes:   0,
	}
	
	// Initialize lanes
	game.lanes[0] = Lane{X: 100, Width: LANE_WIDTH, KeyCode: rl.KeyA}  // A key
	game.lanes[1] = Lane{X: 300, Width: LANE_WIDTH, KeyCode: rl.KeyW}  // W key  
	game.lanes[2] = Lane{X: 500, Width: LANE_WIDTH, KeyCode: rl.KeyD}  // D key
	
	return game
}

// LoadMIDITrack loads notes from the MIDI processor
func (g *Game) LoadMIDITrack(midiProcessor *MIDIProcessor) error {
	g.midiProcessor = midiProcessor
	
	// Find guitar track
	guitarTrack, err := midiProcessor.FindGuitarTrack()
	if err != nil {
		return err
	}
	
	// Find the earliest note to offset timing
	earliestNoteTime := float64(999999)
	for _, midiNote := range guitarTrack.Notes {
		if midiNote.StartTime < earliestNoteTime {
			earliestNoteTime = midiNote.StartTime
		}
	}
	
	fmt.Printf("Earliest note starts at: %.2fs, offsetting all notes...\n", earliestNoteTime)
	
	// Convert MIDI notes to game notes with time offset
	g.gameNotes = make([]GameNote, 0)
	maxTime := 0.0
	
	for _, midiNote := range guitarTrack.Notes {
		// Offset all notes so the first note starts at time 2.0 (giving 2 seconds to get ready)
		adjustedStartTime := midiNote.StartTime - earliestNoteTime + 2.0
		
		// Only include notes that start within the game duration window
		if adjustedStartTime > GAME_DURATION {
			continue // Skip notes that start after the 30-second game window
		}
		
		// Limit note duration so it doesn't extend past game end
		adjustedDuration := midiNote.Duration
		noteEndTime := adjustedStartTime + adjustedDuration
		if noteEndTime > GAME_DURATION {
			adjustedDuration = GAME_DURATION - adjustedStartTime
		}
		
		gameNote := GameNote{
			StartTime: adjustedStartTime,
			Duration:  adjustedDuration,
			Lane:      midiNote.Lane,
			Width:     LANE_WIDTH - 20, // Leave some margin
			Height:    NOTE_HEIGHT,
			IsActive:  true,
			IsHit:     false,
		}
		
		// Calculate song duration (capped at GAME_DURATION)
		if noteEndTime > maxTime {
			maxTime = noteEndTime
		}
		
		g.gameNotes = append(g.gameNotes, gameNote)
	}
	
	g.songDuration = GAME_DURATION // Set to exactly 30 seconds
	g.totalNotes = int32(len(g.gameNotes))
	
	fmt.Printf("Loaded %d game notes from guitar track, song duration: %.1fs\n", 
		len(g.gameNotes), g.songDuration)
	
	// Load audio track for playback
	if g.audioManager != nil {
		// Create adjusted MIDI notes for audio synthesis
		audioNotes := make([]MIDINote, 0)
		for _, midiNote := range guitarTrack.Notes {
			adjustedNote := midiNote
			adjustedNote.StartTime = midiNote.StartTime - earliestNoteTime + 2.0
			
			// Only include notes within the game duration
			if adjustedNote.StartTime <= GAME_DURATION {
				// Limit duration to game end
				if adjustedNote.StartTime + adjustedNote.Duration > GAME_DURATION {
					adjustedNote.Duration = GAME_DURATION - adjustedNote.StartTime
				}
				audioNotes = append(audioNotes, adjustedNote)
			}
		}
		
		err = g.audioManager.LoadMIDITrack(audioNotes)
		if err != nil {
			fmt.Printf("Warning: Failed to load audio track: %v\n", err)
		}
	}
	
	return nil
}

// StartGame starts the game
func (g *Game) StartGame() {
	g.state = StatePlaying
	g.gameStartTime = time.Now()
	g.currentTime = 0
	g.score = 0
	g.combo = 0
	g.maxCombo = 0
	g.perfectHits = 0
	g.goodHits = 0
	g.okHits = 0
	g.missedHits = 0
	
	// Reset all notes
	for i := range g.gameNotes {
		g.gameNotes[i].IsActive = true
		g.gameNotes[i].IsHit = false
	}
	
	// Start audio playback
	if g.audioManager != nil {
		err := g.audioManager.StartPlayback()
		if err != nil {
			fmt.Printf("Warning: Failed to start audio playback: %v\n", err)
		}
	}
	
	fmt.Println("Game started!")
}

// IsPlaying returns whether the game is currently playing
func (g *Game) IsPlaying() bool {
	return g.state == StatePlaying
}

// IsGameOver returns whether the game is over
func (g *Game) IsGameOver() bool {
	return g.state == StateGameOver
}

// EndGame ends the game and transitions to game over state
func (g *Game) EndGame() {
	g.state = StateGameOver
	
	// Stop audio playback
	if g.audioManager != nil {
		g.audioManager.StopPlayback()
	}
	
	fmt.Printf("Game ended! Final score: %d, Max combo: %d\n", g.score, g.maxCombo)
}

// Update updates the game state
func (g *Game) Update(deltaTime float32) {
	if !g.IsPlaying() {
		return
	}
	
	// Update current time
	g.currentTime = time.Since(g.gameStartTime).Seconds()
	
	// Update audio manager
	if g.audioManager != nil {
		g.audioManager.Update()
	}
	
	// Check if song is finished
	if g.currentTime > g.songDuration {
		g.EndGame()
		return
	}
	
	// Update input
	g.updateInput()
	
	// Update notes
	g.updateNotes(deltaTime)
	
	// Update sustained notes
	g.updateSustainedNotes()
	
	// Check for missed notes
	g.checkMissedNotes()
	
	// Check if all notes are processed
	g.checkAllNotesProcessed()
}

// checkAllNotesProcessed checks if all notes have been hit or missed
func (g *Game) checkAllNotesProcessed() {
	processedNotes := g.perfectHits + g.goodHits + g.okHits + g.missedHits
	if processedNotes >= g.totalNotes {
		fmt.Printf("All notes processed: %d/%d\n", processedNotes, g.totalNotes)
		g.EndGame()
	}
}

// updateInput handles keyboard input
func (g *Game) updateInput() {
	for i := range g.lanes {
		lane := &g.lanes[i]
		lane.IsPressed = rl.IsKeyDown(lane.KeyCode)
		
		// Check for key press events
		if rl.IsKeyPressed(lane.KeyCode) {
			g.handleKeyPress(i)
		}
		
		// Check for key release events
		if rl.IsKeyReleased(lane.KeyCode) {
			g.handleKeyRelease(i)
		}
	}
}

// updateNotes updates the position of all notes
func (g *Game) updateNotes(deltaTime float32) {
	for i := range g.gameNotes {
		note := &g.gameNotes[i]
		if !note.IsActive {
			continue
		}
		
		// Calculate note position based on timing
		timeUntilHit := note.StartTime - g.currentTime
		note.Y = g.hitLine - float32(timeUntilHit*NOTE_SPEED)
		
		// Remove notes that are off screen
		if note.Y > float32(g.screenHeight)+50 {
			note.IsActive = false
		}
	}
}

// updateSustainedNotes updates the state of sustained notes being held
func (g *Game) updateSustainedNotes() {
	for i := range g.gameNotes {
		note := &g.gameNotes[i]
		if !note.IsActive || !note.IsPressed || note.IsHit {
			continue
		}
		
		// Check if the key is still being held for this note's lane
		lanePressed := g.lanes[note.Lane].IsPressed
		
		if lanePressed {
			// Update sustain progress based on how far through the note we are
			noteElapsed := g.currentTime - note.StartTime
			note.SustainProgress = noteElapsed / note.Duration
			
			// Clamp to valid range
			if note.SustainProgress < 0 {
				note.SustainProgress = 0
			} else if note.SustainProgress > 1.0 {
				note.SustainProgress = 1.0
			}
			
			// Keep the note marked as being held correctly
			note.IsBeingHeld = true
			
			// Note is being sustained correctly
			
			// Check if the note should be automatically completed
			noteEndTime := note.StartTime + note.Duration
			if g.currentTime >= noteEndTime {
				// Note duration has elapsed, auto-complete it
				note.IsPressed = false
				note.IsBeingHeld = false
				note.IsHit = true
				
				// Award score based on how well it was held
				accuracy := note.HitAccuracy
				g.addScore(accuracy)
				
				// Bonus for sustained notes
				if note.SustainProgress > 0.8 {
					bonusPoints := int32(50 * note.SustainProgress)
					g.score += bonusPoints
				}
				
				// Auto-completed sustained note
			}
		} else {
			// Key released too early, mark as missed
			note.IsPressed = false
			note.IsBeingHeld = false
			note.IsHit = true
			note.HitAccuracy = Miss
			g.addScore(Miss)
			// Sustained note released too early
		}
	}
}

// isSustainedNote checks if a note is a sustained note (duration > 0.3 seconds)
func (g *Game) isSustainedNote(note *GameNote) bool {
	return note.Duration > 0.3
}

// handleKeyPress handles when a key is pressed
func (g *Game) handleKeyPress(laneIndex int) {
	// Find the closest note in this lane
	closestNote := g.findClosestNote(laneIndex)
	if closestNote == nil {
		return
	}
	
	// Calculate hit accuracy for the start of the note
	timeDiff := g.currentTime - closestNote.StartTime
	accuracy := g.calculateAccuracy(timeDiff)
	
	if accuracy != Miss {
		if g.isSustainedNote(closestNote) {
			// For sustained notes, mark as pressed and start tracking
			closestNote.IsPressed = true
			closestNote.PressStartTime = g.currentTime
			closestNote.IsBeingHeld = true
			closestNote.HitAccuracy = accuracy
			// Sustained note started silently
		} else {
			// For short notes, score immediately
			closestNote.IsHit = true
			closestNote.HitAccuracy = accuracy
			g.addScore(accuracy)
			fmt.Printf("Hit! Lane: %d, Accuracy: %v, Score: %d\n", laneIndex, accuracy, g.score)
		}
	}
}

// handleKeyRelease handles when a key is released
func (g *Game) handleKeyRelease(laneIndex int) {
	// Find any sustained notes currently being held in this lane
	for i := range g.gameNotes {
		note := &g.gameNotes[i]
		if !note.IsActive || note.Lane != laneIndex || !note.IsPressed || note.IsHit {
			continue
		}
		
		// Check if this is a proper release at the end of a sustained note
		noteEndTime := note.StartTime + note.Duration
		releaseTimeDiff := g.currentTime - noteEndTime
		releaseAccuracy := g.calculateAccuracy(releaseTimeDiff)
		
		// Complete the sustained note
		note.IsPressed = false
		note.IsBeingHeld = false
		note.IsHit = true
		
		// Calculate final score based on start accuracy and release accuracy
		finalAccuracy := note.HitAccuracy // Start with the initial hit accuracy
		if releaseAccuracy == Miss {
			// Poor release timing, downgrade the score
			if finalAccuracy == Perfect {
				finalAccuracy = Good
			} else if finalAccuracy == Good {
				finalAccuracy = OK
			}
		}
		
		// Award points for completing the sustained note
		g.addScore(finalAccuracy)
		
		// Bonus points for sustained notes held correctly
		if note.SustainProgress > 0.8 { // If held for at least 80% of duration
			bonusPoints := int32(50 * note.SustainProgress)
			g.score += bonusPoints
		}
		
		// Sustained note completed
		break // Only handle one note per release
	}
}

// findClosestNote finds the closest unhit note in the specified lane
func (g *Game) findClosestNote(laneIndex int) *GameNote {
	var closestNote *GameNote
	minDistance := float64(1000000) // Large number
	
	for i := range g.gameNotes {
		note := &g.gameNotes[i]
		// FIXED: Also exclude notes that are already being pressed (sustained notes)
		if !note.IsActive || note.IsHit || note.IsPressed || note.Lane != laneIndex {
			continue
		}
		
		distance := note.StartTime - g.currentTime
		if distance < minDistance && distance > -0.2 { // Allow 200ms window after note
			minDistance = distance
			closestNote = note
		}
	}
	
	return closestNote
}

// calculateAccuracy calculates hit accuracy based on timing difference
func (g *Game) calculateAccuracy(timeDiff float64) HitAccuracy {
	absTimeDiff := timeDiff
	if absTimeDiff < 0 {
		absTimeDiff = -absTimeDiff
	}
	
	if absTimeDiff <= 0.05 { // 50ms
		return Perfect
	} else if absTimeDiff <= 0.1 { // 100ms
		return Good
	} else if absTimeDiff <= 0.15 { // 150ms
		return OK
	} else {
		return Miss
	}
}

// addScore adds score based on hit accuracy
func (g *Game) addScore(accuracy HitAccuracy) {
	switch accuracy {
	case Perfect:
		g.score += 100
		g.combo++
		g.perfectHits++
	case Good:
		g.score += 75
		g.combo++
		g.goodHits++
	case OK:
		g.score += 50
		g.combo++
		g.okHits++
	case Miss:
		g.combo = 0
		g.missedHits++
	}
	
	// Update max combo
	if g.combo > g.maxCombo {
		g.maxCombo = g.combo
	}
	
	// Combo bonus
	if g.combo > 10 {
		g.score += int32(g.combo / 10)
	}
}

// checkMissedNotes checks for notes that were missed
func (g *Game) checkMissedNotes() {
	for i := range g.gameNotes {
		note := &g.gameNotes[i]
		if !note.IsActive || note.IsHit {
			continue
		}
		
		// If note is too far past the hit line, mark as missed
		if g.currentTime > note.StartTime+0.2 { // 200ms grace period
			note.IsHit = true
			note.HitAccuracy = Miss
			g.addScore(Miss)
			fmt.Printf("Missed note in lane %d\n", note.Lane)
		}
	}
}