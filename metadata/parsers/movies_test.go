package parsers

import (
	"testing"
)

func TestParseMovieName(t *testing.T) {
	tests := make(map[string]ParsedMovieInfo)
	tests["Mad.Max.Fury.Road.(2015).mkv"] = ParsedMovieInfo{Year: 2015, Title: "Mad Max Fury Road"}
	tests["The Matrix Revolutions (2003).mkv"] = ParsedMovieInfo{Year: 2003, Title: "The Matrix Revolutions"}

	for name, mi := range tests {
		t.Log("running test on:", name)
		newMi := ParseMovieName(name)
		if newMi.Year != mi.Year {
			t.Errorf("Year %v did not match expected year %v\n", newMi.Year, mi.Year)
		}

		if newMi.Title != mi.Title {
			t.Errorf("Title %v did not match expected Title %v\n", newMi.Title, mi.Title)
		}
	}
}
