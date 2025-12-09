package api

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"radikojp/model"
)

const (
	StationListURLFmt = "https://api.radiko.jp/program/v3/now/%s.xml"
	StreamURLFmt      = "https://radiko.jp/v3/station/stream/pc_html5/%s.xml"
)

// GetStations 获取指定地区的电台列表
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
