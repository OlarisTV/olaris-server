package parsers

import (
	"testing"
)

func TestParseSerieName(t *testing.T) {
	tests := make(map[string]ParsedSeriesInfo)
	tests["Battlestar Galactica - S01E04 (1978)"] = ParsedSeriesInfo{Year: 1978, Title: "Battlestar Galactica", EpisodeNum: 4, SeasonNum: 1}
	tests["Battlestar Galactica - S02E03"] = ParsedSeriesInfo{Year: 0, Title: "Battlestar Galactica", EpisodeNum: 3, SeasonNum: 2}
	tests["Battlestar Galactica - S2E3"] = ParsedSeriesInfo{Year: 0, Title: "Battlestar Galactica", EpisodeNum: 3, SeasonNum: 2}
	tests["This does not Exist"] = ParsedSeriesInfo{Year: 0, Title: "This does not Exist", EpisodeNum: 0, SeasonNum: 0}
	tests["Battlestar.Galactica.-.S02E03.mkv"] = ParsedSeriesInfo{Year: 0, Title: "Battlestar Galactica", EpisodeNum: 3, SeasonNum: 2}
	tests["Angel.3x2"] = ParsedSeriesInfo{Year: 0, Title: "Angel", EpisodeNum: 2, SeasonNum: 3}

	for name, mi := range tests {
		t.Log("running test on:", name)
		newMi := ParseSerieName(name)
		if newMi.Year != mi.Year {
			t.Errorf("Year %v did not match expected year %v\n", newMi.Year, mi.Year)
		}

		if newMi.EpisodeNum != mi.EpisodeNum {
			t.Errorf("Episode %v did not match expected episode %v\n", newMi.EpisodeNum, mi.EpisodeNum)
		}

		if newMi.Title != mi.Title {
			t.Errorf("Title %v did not match expected Title %v\n", newMi.Title, mi.Title)
		}
	}
}
