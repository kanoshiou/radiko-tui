package main

import "fmt"

// Configuration example - modify these values as needed

// StationConfig represents station configuration
type StationConfig struct {
	Name      string
	StationID string
}

// Stations Predefined station list
var Stations = map[string]StationConfig{
	"QRR": {
		Name:      "文化放送",
		StationID: "QRR",
	},
	"TBS": {
		Name:      "TBS ラジオ",
		StationID: "TBS",
	},
	"LFR": {
		Name:      "ニッポン放送",
		StationID: "LFR",
	},
	"INT": {
		Name:      "interfm",
		StationID: "INT",
	},
	"FMT": {
		Name:      "TOKYO FM",
		StationID: "FMT",
	},
	"FMJ": {
		Name:      "J-WAVE",
		StationID: "FMJ",
	},
}

// GetStationURL generates a full URL based on the station ID
func GetStationURL(stationID string, lsid string) string {
	baseURL := "https://c-radiko.smartstream.ne.jp/%s/_definst_/simul-stream.stream/playlist.m3u8"
	return fmt.Sprintf(baseURL+"?station_id=%s&l=30&lsid=%s&type=b", stationID, stationID, lsid)
}
