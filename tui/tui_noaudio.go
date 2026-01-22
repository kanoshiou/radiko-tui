//go:build noaudio

package tui

import (
	"fmt"

	"radiko-tui/config"
	"radiko-tui/model"
)

// Run is a stub that returns an error for noaudio builds
// The TUI requires audio support and is not available in server-only mode
func Run(stations []model.Station, authToken string, cfg config.Config, serverURL string) error {
	return fmt.Errorf("TUI モードは noaudio ビルドではサポートされていません。--server フラグを使用してください")
}
