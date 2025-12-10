package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"radiko-tui/model"
)

const (
	StationListURLFmt = "https://api.radiko.jp/program/v3/now/%s.xml"
	StreamURLFmt      = "https://radiko.jp/v3/station/stream/pc_html5/%s.xml"
)

// GetStations retrieves the list of stations for a specified area
func GetStations(areaID string) ([]model.Station, error) {
	url := fmt.Sprintf(StationListURLFmt, areaID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch station list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch station list: status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var radikoStations model.RadikoStations
	if err := xml.Unmarshal(data, &radikoStations); err != nil {
		return nil, fmt.Errorf("failed to parse station list XML: %w", err)
	}

	return radikoStations.Stations, nil
}

func GetStreamURLs(stationID string) ([]string, error) {
	url := fmt.Sprintf(StreamURLFmt, stationID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stream URL for station %s: %w", stationID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch stream URL: status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var radikoURLs model.RadikoURLs
	if err := xml.Unmarshal(data, &radikoURLs); err != nil {
		return nil, fmt.Errorf("failed to parse stream URL XML: %w", err)
	}

	if len(radikoURLs.URLs) == 0 {
		return nil, fmt.Errorf("no stream URLs found for station %s", stationID)
	}

	var urls []string
	for _, u := range radikoURLs.URLs {
		if u.PlaylistCreateURL != "" {
			urls = append(urls, u.PlaylistCreateURL)
		}
	}

	return urls, nil
}

// ProgramURLFmt is the program info API URL format
const ProgramURLFmt = "https://api.radiko.jp/program/v4/date/%s/station/%s.json"

// GetCurrentProgram retrieves the current program for a station
func GetCurrentProgram(stationID string) (*model.Program, error) {
	// Use Japan timezone (UTC+9)
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)
	dateStr := now.Format("20060102")
	timeStr := now.Format("20060102150405")

	url := fmt.Sprintf(ProgramURLFmt, dateStr, stationID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var progResp model.ProgramResponse
	if err := json.Unmarshal(data, &progResp); err != nil {
		return nil, err
	}

	// Find the program for the current time
	for _, station := range progResp.Stations {
		if station.StationID == stationID {
			for _, prog := range station.Programs.Program {
				// Check if current time is within the program's time range
				if prog.Ft <= timeStr && timeStr < prog.To {
					return &prog, nil
				}
			}
		}
	}

	return nil, nil
}
