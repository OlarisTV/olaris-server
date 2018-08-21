package parsers

import (
	"testing"
)

func TestParseMovieName(t *testing.T) {
	tests := make(map[string]ParsedMovieInfo)
	tests["Mad.Max.Fury.Road.(2015).mkv"] = ParsedMovieInfo{Year: 2015, Title: "Mad Max Fury Road"}
	tests["The Matrix Revolutions (2003).mkv"] = ParsedMovieInfo{Year: 2003, Title: "The Matrix Revolutions"}
	tests["The.Matrix.(1999).mkv"] = ParsedMovieInfo{Year: 1999, Title: "The Matrix"}
	tests["The.Matrix.mkv"] = ParsedMovieInfo{Year: 0, Title: "The Matrix"}
	tests["The.Matrix.1999.mkv"] = ParsedMovieInfo{Year: 1999, Title: "The Matrix"}
	tests["300.mkv"] = ParsedMovieInfo{Year: 0, Title: "300"}
	tests["300 (2006).mkv"] = ParsedMovieInfo{Year: 2006, Title: "300"}

	for name, mi := range tests {
		t.Log("running test on:", name)
		newMi := ParseMovieName(name)
		if newMi.Year != mi.Year {
			t.Errorf("Year %v did not match expected year %v\n", newMi.Year, mi.Year)
		}

		if newMi.Title != mi.Title {
			t.Errorf("Title '%v' did not match expected title '%v'\n", newMi.Title, mi.Title)
		}
	}
}
