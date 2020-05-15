package parsers

import (
	"testing"
)

func TestParseSeriesName(t *testing.T) {
	tests := make(map[string]ParsedSeriesInfo)
	tests["Angel.3x2.avi"] = ParsedSeriesInfo{Year: "", Title: "Angel", EpisodeNum: 2, SeasonNum: 3}
	tests["Battlestar Galactica - S01E04 (1978).m4v"] = ParsedSeriesInfo{Year: "1978", Title: "Battlestar Galactica", EpisodeNum: 4, SeasonNum: 1}
	tests["Battlestar.Galactica.1978.S01E04 - The Lost Planet of the Gods (1).mkv"] = ParsedSeriesInfo{Year: "1978", Title: "Battlestar Galactica", EpisodeNum: 4, SeasonNum: 1}
	tests["Battlestar Galactica (1978)/s1/04. The Lost Planet of the Gods part 1.mp4"] = ParsedSeriesInfo{Year: "1978", Title: "Battlestar Galactica", EpisodeNum: 4, SeasonNum: 1}
	tests["Battlestar Galactica (1978)/season 1/04. The Lost Planet of the Gods.mkv"] = ParsedSeriesInfo{Year: "1978", Title: "Battlestar Galactica", EpisodeNum: 4, SeasonNum: 1}
	tests["Battlestar Galactica/s1/04. The Lost Planet of the Gods.mkv"] = ParsedSeriesInfo{Year: "", Title: "Battlestar Galactica", EpisodeNum: 4, SeasonNum: 1}
	tests["Battlestar Galactica (2003) - S02E03.mp4"] = ParsedSeriesInfo{Year: "2003", Title: "Battlestar Galactica", EpisodeNum: 3, SeasonNum: 2}
	tests["Battlestar Galactica - S2E3.wmv"] = ParsedSeriesInfo{Year: "", Title: "Battlestar Galactica", EpisodeNum: 3, SeasonNum: 2}
	tests["Battlestar.Galactica.-.S02E03.mkv"] = ParsedSeriesInfo{Year: "", Title: "Battlestar Galactica", EpisodeNum: 3, SeasonNum: 2}
	tests["Mr. Robot (2016).S01E04 - eps1.3_da3m0ns.mp4.mkv"] = ParsedSeriesInfo{Year: "2016", Title: "Mr Robot", EpisodeNum: 4, SeasonNum: 1}
	tests["Mr. Robot (2016) - S01E04.mpg"] = ParsedSeriesInfo{Year: "2016", Title: "Mr Robot", EpisodeNum: 4, SeasonNum: 1}
	tests["Mr. Robot/Season 2/03.m2ts"] = ParsedSeriesInfo{Year: "", Title: "Mr Robot", EpisodeNum: 3, SeasonNum: 2}
	tests["This does not Exist"] = ParsedSeriesInfo{Year: "", Title: "This does not Exist", EpisodeNum: 0, SeasonNum: 0}

	for name, mi := range tests {
		t.Log("running test on:", name)
		newMi := ParseSeriesName(name)
		if newMi.Year != mi.Year {
			t.Errorf("Year [%v] did not match expected year [%v] for input [%v]\n", newMi.Year, mi.Year, name)
		}

		if newMi.EpisodeNum != mi.EpisodeNum {
			t.Errorf("Episode [%v] did not match expected episode [%v] for input [%v]\n", newMi.EpisodeNum, mi.EpisodeNum, name)
		}

		if newMi.Title != mi.Title {
			t.Errorf("Title [%v] did not match expected title [%v] for input [%v]\n", newMi.Title, mi.Title, name)
		}
	}
}
