package main

import (
	"fmt"
	
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Renderer handles all drawing operations
type Renderer struct {
	game *Game
}

// NewRenderer creates a new renderer
func NewRenderer(game *Game) *Renderer {
	return &Renderer{
		game: game,
	}
}

// Draw renders the entire game
func (r *Renderer) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.Black)
	
	switch r.game.state {
	case StateMenu:
		r.drawMenu()
	case StatePlaying:
		r.drawGameplay()
	case StateGameOver:
		r.drawGameOver()
	}
	
	rl.EndDrawing()
}

// drawGameplay draws the main gameplay screen
func (r *Renderer) drawGameplay() {
	// Draw lanes
	r.drawLanes()
	
	// Draw hit line
	r.drawHitLine()
	
	// Draw notes
	r.drawNotes()
	
	// Draw UI
	r.drawUI()
	
	// Draw progress bar
	r.drawProgressBar()
}

// drawMenu draws the main menu
func (r *Renderer) drawMenu() {
	centerX := r.game.screenWidth / 2
	centerY := r.game.screenHeight / 2
	
	// Title
	title := "Guitar Hero Game"
	titleWidth := rl.MeasureText(title, 40)
	rl.DrawText(title, centerX-titleWidth/2, centerY-100, 40, rl.White)
	
	// Instructions
	instructions := []string{
		"Press SPACE to Start",
		"Use A, W, D keys to hit notes",
		"Hit notes when they reach the red line",
		"Press ESC to quit",
	}
	
	for i, instruction := range instructions {
		textWidth := rl.MeasureText(instruction, 20)
		rl.DrawText(instruction, centerX-textWidth/2, centerY-20+int32(i*30), 20, rl.LightGray)
	}
}

// drawGameOver draws the game over screen
func (r *Renderer) drawGameOver() {
	centerX := r.game.screenWidth / 2
	centerY := r.game.screenHeight / 2
	
	// Game Over title
	title := "Game Over!"
	titleWidth := rl.MeasureText(title, 40)
	rl.DrawText(title, centerX-titleWidth/2, centerY-150, 40, rl.Red)
	
	// Final score
	scoreText := fmt.Sprintf("Final Score: %d", r.game.score)
	scoreWidth := rl.MeasureText(scoreText, 30)
	rl.DrawText(scoreText, centerX-scoreWidth/2, centerY-100, 30, rl.White)
	
	// Max combo
	comboText := fmt.Sprintf("Max Combo: %d", r.game.maxCombo)
	comboWidth := rl.MeasureText(comboText, 25)
	rl.DrawText(comboText, centerX-comboWidth/2, centerY-60, 25, rl.Yellow)
	
	// Statistics
	stats := []string{
		fmt.Sprintf("Perfect: %d", r.game.perfectHits),
		fmt.Sprintf("Good: %d", r.game.goodHits),
		fmt.Sprintf("OK: %d", r.game.okHits),
		fmt.Sprintf("Missed: %d", r.game.missedHits),
	}
	
	for i, stat := range stats {
		statWidth := rl.MeasureText(stat, 20)
		color := rl.White
		switch i {
		case 0:
			color = rl.Gold
		case 1:
			color = rl.Green
		case 2:
			color = rl.Blue
		case 3:
			color = rl.Red
		}
		rl.DrawText(stat, centerX-statWidth/2, centerY+int32(i*25), 20, color)
	}
	
	// Accuracy
	accuracy := float32(0)
	if r.game.totalNotes > 0 {
		accuracy = float32(r.game.perfectHits+r.game.goodHits+r.game.okHits) / float32(r.game.totalNotes) * 100
	}
	accuracyText := fmt.Sprintf("Accuracy: %.1f%%", accuracy)
	accuracyWidth := rl.MeasureText(accuracyText, 25)
	rl.DrawText(accuracyText, centerX-accuracyWidth/2, centerY+120, 25, rl.White)
	
	// Restart instruction
	restartText := "Press SPACE to play again or ESC to quit"
	restartWidth := rl.MeasureText(restartText, 20)
	rl.DrawText(restartText, centerX-restartWidth/2, centerY+170, 20, rl.LightGray)
}

// drawLanes draws the three game lanes
func (r *Renderer) drawLanes() {
	for i, lane := range r.game.lanes {
		// Lane background
		color := rl.DarkGray
		if lane.IsPressed {
			color = rl.Gray
		}
		
		rl.DrawRectangle(
			int32(lane.X),
			0,
			int32(lane.Width),
			r.game.screenHeight,
			color,
		)
		
		// Lane borders
		rl.DrawRectangleLines(
			int32(lane.X),
			0,
			int32(lane.Width),
			r.game.screenHeight,
			rl.White,
		)
		
		// Lane labels
		keyText := []string{"A", "W", "D"}[i]
		textX := int32(lane.X + lane.Width/2 - 10)
		textY := int32(r.game.hitLine + 50)
		rl.DrawText(keyText, textX, textY, 30, rl.White)
	}
}

// drawHitLine draws the horizontal hit line
func (r *Renderer) drawHitLine() {
	rl.DrawLine(
		0,
		int32(r.game.hitLine),
		r.game.screenWidth,
		int32(r.game.hitLine),
		rl.Red,
	)
	
	// Make hit line more visible
	rl.DrawLine(
		0,
		int32(r.game.hitLine-1),
		r.game.screenWidth,
		int32(r.game.hitLine-1),
		rl.Red,
	)
	rl.DrawLine(
		0,
		int32(r.game.hitLine+1),
		r.game.screenWidth,
		int32(r.game.hitLine+1),
		rl.Red,
	)
}

// drawNotes draws all active game notes
func (r *Renderer) drawNotes() {
	for _, note := range r.game.gameNotes {
		if !note.IsActive {
			continue
		}
		
		// Calculate note position
		lane := r.game.lanes[note.Lane]
		noteX := lane.X + 10 // Small margin from lane edge
		noteY := note.Y
		
		// Skip notes that are off screen
		if noteY < -note.Height || noteY > float32(r.game.screenHeight)+note.Height {
			continue
		}
		
		// Choose note color based on state
		var color rl.Color
		if note.IsHit {
			switch note.HitAccuracy {
			case Perfect:
				color = rl.Gold
			case Good:
				color = rl.Green
			case OK:
				color = rl.Blue
			case Miss:
				color = rl.Red
			}
		} else if note.IsBeingHeld {
			// Green for sustained notes being held correctly
			color = rl.Green
		} else if note.IsPressed {
			// Light green for sustained notes just started
			color = rl.Lime
		} else {
			// Different colors for different lanes
			colors := []rl.Color{rl.SkyBlue, rl.Pink, rl.Orange}
			color = colors[note.Lane]
		}
		
		// Draw note
		rl.DrawRectangle(
			int32(noteX),
			int32(noteY),
			int32(note.Width),
			int32(note.Height),
			color,
		)
		
		// Draw note border
		rl.DrawRectangleLines(
			int32(noteX),
			int32(noteY),
			int32(note.Width),
			int32(note.Height),
			rl.White,
		)
		
		// For sustained notes, draw length indicator
		if note.Duration > 0.3 { // Only for sustained notes
			sustainHeight := int32(note.Duration * NOTE_SPEED)
			sustainX := int32(noteX + note.Width/4)
			sustainWidth := int32(note.Width/2)
			
			// Draw the full sustain tail with transparency
			rl.DrawRectangle(
				sustainX,
				int32(noteY + note.Height),
				sustainWidth,
				sustainHeight,
				rl.ColorAlpha(color, 0.3),
			)
			
			// If note is being held, show progress
			if note.IsPressed && note.SustainProgress > 0 {
				progressHeight := int32(float64(sustainHeight) * note.SustainProgress)
				rl.DrawRectangle(
					sustainX,
					int32(noteY + note.Height),
					sustainWidth,
					progressHeight,
					rl.ColorAlpha(rl.Green, 0.7),
				)
			}
		}
	}
}

// drawUI draws the game UI (score, combo, etc.)
func (r *Renderer) drawUI() {
	// Score
	scoreText := fmt.Sprintf("Score: %d", r.game.score)
	rl.DrawText(scoreText, 10, 10, 20, rl.White)
	
	// Combo
	if r.game.combo > 0 {
		comboText := fmt.Sprintf("Combo: %d", r.game.combo)
		rl.DrawText(comboText, 10, 40, 20, rl.Yellow)
	}
	
	// Time remaining (show countdown)
	timeRemaining := r.game.songDuration - r.game.currentTime
	if timeRemaining < 0 {
		timeRemaining = 0
	}
	timeText := fmt.Sprintf("Time: %.1fs", timeRemaining)
	timeColor := rl.LightGray
	if timeRemaining < 5.0 {
		timeColor = rl.Red // Red when less than 5 seconds remain
	}
	rl.DrawText(timeText, 10, 70, 20, timeColor)
	
	// Audio status indicator
	if r.game.audioManager != nil {
		audioText := "♪ Audio: "
		if r.game.audioManager.IsPlaying() {
			audioText += "ON"
			rl.DrawText(audioText, 10, 100, 16, rl.Green)
		} else {
			audioText += "OFF"
			rl.DrawText(audioText, 10, 100, 16, rl.Red)
		}
	}
}

// drawProgressBar draws the song progress bar
func (r *Renderer) drawProgressBar() {
	if r.game.songDuration <= 0 {
		return
	}
	
	barWidth := int32(300)
	barHeight := int32(10)
	barX := r.game.screenWidth - barWidth - 20
	barY := int32(20)
	
	// Background
	rl.DrawRectangle(barX, barY, barWidth, barHeight, rl.DarkGray)
	
	// Progress
	progress := float32(r.game.currentTime / r.game.songDuration)
	if progress > 1.0 {
		progress = 1.0
	}
	progressWidth := int32(float32(barWidth) * progress)
	rl.DrawRectangle(barX, barY, progressWidth, barHeight, rl.Green)
	
	// Border
	rl.DrawRectangleLines(barX, barY, barWidth, barHeight, rl.White)
	
	// Time text (remaining/total)
	timeRemaining := r.game.songDuration - r.game.currentTime
	if timeRemaining < 0 {
		timeRemaining = 0
	}
	timeText := fmt.Sprintf("%.1fs remaining", timeRemaining)
	rl.DrawText(timeText, barX, barY+barHeight+5, 16, rl.White)
}

// drawInstructions draws game instructions
func (r *Renderer) drawInstructions() {
	instructions := []string{
		"Use A, W, D keys to hit the notes",
		"Hit notes when they reach the red line",
		"Hold for sustained notes",
		"Press SPACE to start/pause",
		"Press ESC to quit",
	}
	
	startY := int32(r.game.screenHeight - int32(len(instructions)*20) - 10)
	
	for i, instruction := range instructions {
		rl.DrawText(
			instruction,
			10,
			startY+int32(i*20),
			16,
			rl.LightGray,
		)
	}
}