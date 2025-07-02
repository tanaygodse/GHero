package main

import (
	"fmt"
	"sort"
)

// DebugNotes prints information about the first few notes for debugging
func DebugNotes(notes []MIDINote) {
	if len(notes) == 0 {
		fmt.Println("No notes to debug")
		return
	}
	
	// Sort notes by start time
	sortedNotes := make([]MIDINote, len(notes))
	copy(sortedNotes, notes)
	sort.Slice(sortedNotes, func(i, j int) bool {
		return sortedNotes[i].StartTime < sortedNotes[j].StartTime
	})
	
	fmt.Printf("=== NOTE TIMING DEBUG ===\n")
	fmt.Printf("Total notes: %d\n", len(notes))
	fmt.Printf("First 10 notes:\n")
	
	for i := 0; i < 10 && i < len(sortedNotes); i++ {
		note := sortedNotes[i]
		fmt.Printf("Note %d: Start=%.2fs, Duration=%.2fs, Pitch=%d, Lane=%d\n", 
			i+1, note.StartTime, note.Duration, note.Pitch, note.Lane)
	}
	
	fmt.Printf("\nTime ranges:\n")
	fmt.Printf("Earliest note: %.2fs\n", sortedNotes[0].StartTime)
	fmt.Printf("Latest note: %.2fs\n", sortedNotes[len(sortedNotes)-1].StartTime)
	
	// Count notes in first minute
	notesInFirstMinute := 0
	for _, note := range sortedNotes {
		if note.StartTime <= 60.0 {
			notesInFirstMinute++
		} else {
			break
		}
	}
	fmt.Printf("Notes in first 60 seconds: %d\n", notesInFirstMinute)
	
	// Find first note in each lane
	laneFirstNotes := [3]float64{-1, -1, -1}
	for _, note := range sortedNotes {
		if note.Lane >= 0 && note.Lane < 3 && laneFirstNotes[note.Lane] == -1 {
			laneFirstNotes[note.Lane] = note.StartTime
		}
	}
	
	fmt.Printf("First note per lane:\n")
	laneNames := []string{"A", "W", "D"}
	for i, startTime := range laneFirstNotes {
		if startTime >= 0 {
			fmt.Printf("Lane %s: %.2fs\n", laneNames[i], startTime)
		} else {
			fmt.Printf("Lane %s: No notes\n", laneNames[i])
		}
	}
	fmt.Printf("=========================\n")
}