package main

import (
	"fmt"
	"log"
	
	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	fmt.Println("Guitar Hero Game - Starting...")
	
	// Initialize MIDI processor
	midiProcessor := NewMIDIProcessor()
	
	// Load and analyze the test MIDI file
	err := midiProcessor.LoadMIDI("assets/test.mid")
	if err != nil {
		log.Fatalf("Failed to load MIDI file: %v", err)
	}
	
	// Analyze tracks to find guitar track
	guitarTrack, err := midiProcessor.FindGuitarTrack()
	if err != nil {
		log.Fatalf("Failed to find guitar track: %v", err)
	}
	
	fmt.Printf("Found guitar track with %d notes\n", len(guitarTrack.Notes))
	
	// Initialize Raylib
	rl.InitWindow(SCREEN_WIDTH, SCREEN_HEIGHT, "Guitar Hero Game")
	defer rl.CloseWindow()
	
	rl.SetTargetFPS(60)
	
	// Initialize game
	game := NewGame()
	err = game.LoadMIDITrack(midiProcessor)
	if err != nil {
		log.Fatalf("Failed to load MIDI track: %v", err)
	}
	
	// Ensure audio cleanup on exit
	defer func() {
		if game.audioManager != nil {
			game.audioManager.Cleanup()
		}
	}()
	
	// Initialize renderer
	renderer := NewRenderer(game)
	
	fmt.Println("Guitar Hero Game - Ready to start!")
	
	// Main game loop
	for !rl.WindowShouldClose() {
		// Handle input based on game state
		if rl.IsKeyPressed(rl.KeySpace) {
			switch game.state {
			case StateMenu:
				game.StartGame()
			case StatePlaying:
				// Pause functionality removed for simplicity
				// You can add pause state if needed
			case StateGameOver:
				game.state = StateMenu // Return to menu for restart
			}
		}
		
		if rl.IsKeyPressed(rl.KeyEscape) {
			break
		}
		
		// Update game
		deltaTime := rl.GetFrameTime()
		game.Update(deltaTime)
		
		// Render
		renderer.Draw()
	}
	
	fmt.Println("Guitar Hero Game - Goodbye!")
}