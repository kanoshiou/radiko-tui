package player

import "time"

// Player defines the interface for audio playback
type Player interface {
	Play(urlOrID string) error
	Stop()
	IsPlaying() bool

	SetVolume(volume float64)
	GetVolume() float64
	IncreaseVolume(delta float64)
	DecreaseVolume(delta float64)
	ToggleMute()
	IsMuted() bool

	Reconnect() error

	// Recording methods
	StartRecording(stationName string) error
	StopRecording() (string, error)
	IsRecording() bool
	GetRecordingInfo() (filePath string, duration time.Duration, stationName string)
	ToggleRecording(stationName string) (started bool, filePath string, err error)
}
