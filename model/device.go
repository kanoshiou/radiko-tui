package model

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// RandomDeviceInfo represents random device information
type RandomDeviceInfo struct {
	AppVersion string // App version
	UserID     string // User ID
	UserAgent  string // User-Agent
	Device     string // Device identifier
}

// VersionInfo contains Android SDK and build version information
type VersionInfo struct {
	SDK    string
	Builds []string
}

// VERSION_MAP contains Android version mappings
var VERSION_MAP = map[string]VersionInfo{
	"7.0.0": {
		SDK: "24",
		Builds: []string{
			"NBD92Q", "NBD92N", "NBD92G", "NBD92F", "NBD92E", "NBD92D",
			"NBD91Z", "NBD91Y", "NBD91X", "NBD91U", "N5D91L", "NBD91P",
			"NRD91K", "NRD91N", "NBD90Z", "NBD90X", "NBD90W", "NRD91D",
			"NRD90U", "NRD90T", "NRD90S", "NRD90R", "NRD90M",
		},
	},
	"7.1.0": {
		SDK:    "25",
		Builds: []string{"NDE63X", "NDE63V", "NDE63U", "NDE63P", "NDE63L", "NDE63H"},
	},
	"7.1.1": {
		SDK: "25",
		Builds: []string{
			"N9F27M", "NGI77B", "N6F27M", "N4F27P", "N9F27L", "NGI55D",
			"N4F27O", "N8I11B", "N9F27H", "N6F27I", "N4F27K", "N9F27F",
			"N6F27H", "N4F27I", "N9F27C", "N6F27E", "N4F27E", "N6F27C",
			"N4F27B", "N6F26Y", "NOF27D", "N4F26X", "N4F26U", "N6F26U",
			"NUF26N", "NOF27C", "NOF27B", "N4F26T", "NMF27D", "NMF26X",
			"NOF26W", "NOF26V", "N6F26R", "NUF26K", "N4F26Q", "N4F26O",
			"N6F26Q", "N4F26M", "N4F26J", "N4F26I", "NMF26V", "NMF26U",
			"NMF26R", "NMF26Q", "NMF26O", "NMF26J", "NMF26H", "NMF26F",
		},
	},
	"7.1.2": {
		SDK: "25",
		Builds: []string{
			"N2G48H", "NZH54D", "NKG47S", "NHG47Q", "NJH47F", "N2G48C",
			"NZH54B", "NKG47M", "NJH47D", "NHG47O", "N2G48B", "N2G47Z",
			"NJH47B", "NJH34C", "NKG47L", "NHG47N", "N2G47X", "N2G47W",
			"NHG47L", "N2G47T", "N2G47R", "N2G47O", "NHG47K", "N2G47J",
			"N2G47H", "N2G47F", "N2G47E", "N2G47D",
		},
	},
	"8.0.0": {
		SDK:    "26",
		Builds: []string{"5650811", "5796467", "5948681", "6107732", "6127070"},
	},
	"8.1.0": {
		SDK:    "27",
		Builds: []string{"5794017", "6107733", "6037697"},
	},
	"9.0.0": {
		SDK:    "28",
		Builds: []string{"5948683", "5794013", "6127072"},
	},
	"10.0.0": {
		SDK:    "29",
		Builds: []string{"5933585", "6969601", "7023426", "7070703"},
	},
	"11.0.0": {
		SDK:    "30",
		Builds: []string{"RP1A.201005.006", "RQ1A.201205.011", "RQ1A.210105.002"},
	},
	"12.0.0": {
		SDK: "31",
		Builds: []string{
			"SD1A.210817.015.A4", "SD1A.210817.019.B1",
			"SD1A.210817.037", "SQ1D.220105.007",
		},
	},
	"13.0.0": {
		SDK: "33",
		Builds: []string{
			"TQ3C.230805.001.B2", "TQ3A.230805.001.A2", "TQ3A.230705.001.A1",
			"TQ2B.230505.005.A1", "TQ2A.230505.002", "TQ2A.230405.003.E1",
		},
	},
}

// MODEL_LIST contains device model list
var MODEL_LIST = []string{
	// Samsung Galaxy S7 Edge
	"SC-02H", "SCV33", "SM-G935F", "SM-G935X", "SM-G935W8", "SM-G935K",
	"SM-G935L", "SM-G935S", "SAMSUNG-SM-G935A", "SM-G935VC", "SM-G9350",
	"SM-G935P", "SM-G935T", "SM-G935U", "SM-G935R4", "SM-G935V",
	// Samsung Galaxy S8
	"SC-02J", "SCV36", "SM-G950F", "SM-G950N", "SM-G950W", "SM-G9500",
	"SM-G9508", "SM-G950U", "SM-G950U1", "SM-G892A", "SM-G892U",
	// Samsung Galaxy S8+
	"SC-03J", "SCV35", "SM-G955F", "SM-G955N", "SM-G955W", "SM-G9550",
	"SM-G955U", "SM-G955U1",
	// Samsung Galaxy S9
	"SM-G960F", "SM-G960N", "SM-G9600", "SM-G9608", "SM-G960W",
	"SM-G960U", "SM-G960U1",
	// Samsung Galaxy S9+
	"SM-G965F", "SM-G965N", "SM-G9650", "SM-G965W", "SM-G965U", "SM-G965U1",
	// Samsung Galaxy Note 7
	"SC-01J", "SCV34", "SM-N930F", "SM-N930X", "SM-N930K", "SM-N930L",
	"SM-N930S", "SM-N930R7", "SAMSUNG-SM-N930A", "SM-N930W8", "SM-N9300",
	"SGH-N037", "SM-N930R6", "SM-N930P", "SM-N930VL", "SM-N930T",
	"SM-N930U", "SM-N930R4", "SM-N930V",
	// Samsung Galaxy Note 8
	"SC-01K", "SCV37", "SM-N950F", "SM-N950N", "SM-N950XN", "SM-N950U",
	"SM-N9500", "SM-N9508", "SM-N950W", "SM-N950U1",
	// Kyocera
	"WX06K", "404KC", "503KC", "602KC", "KYV32", "E6782", "KYL22",
	"WX04K", "KYV36", "KYL21", "302KC", "KYV42", "KYV37", "C5155",
	"SKT01", "KYY24", "KYV35", "KYV41", "E6715", "KYY21", "KYY22",
	"KYY23", "KYV31", "KYV34", "KYV38", "WX10K", "KYL23", "KYV39", "KYV40",
	// Sony Xperia
	"C6902", "C6903", "C6906", "C6916", "C6943", "L39h", "L39t", "L39u",
	"SO-01F", "SOL23", "D5503", "M51w", "SO-02F", "D6502", "D6503",
	"D6543", "SO-03F",
	// Sharp
	"605SH", "SH-03J", "SHV39", "701SH", "SH-M06",
	// Fujitsu Arrows
	"101F", "201F", "202F", "301F", "IS12F", "F-03D", "F-03E", "M01",
	"M305", "M357", "M555", "F-11D", "F-06E", "EM01F", "F-05E", "FJT21",
	"F-01D", "FAR70B", "FAR7", "F-04E", "F-02E", "F-10D", "F-05D",
	"FJL22", "ISW11F", "ISW13F", "FJL21", "F-074", "F-07D",
	// Google Pixel
	"G9FPL", "GWKK3", "GHL1X", "G0DZQ", "G82U8", "GP4BC", "GE2AE",
	"GVU6C", "GQML3", "GX7AS", "GB62Z", "G1AZG", "GLUOG", "G8VOU",
	"GB7N6", "G9S9B16", "G1F8F", "G4S1M", "GD1YQ", "GTT9Q",
}

// APP_VERSIONS contains Radiko app version list
var APP_VERSIONS = []string{
	"8.0.11", "8.0.10", "8.0.9", "8.0.7", "8.0.6", "8.0.5", "8.0.4",
	"8.0.3", "8.0.2", "7.5.7", "7.5.6", "7.5.5", "7.5.0", "7.4.17",
	"7.4.16", "7.4.15", "7.4.14", "7.4.13", "7.4.12", "7.4.11",
	"7.4.10", "7.4.5", "7.4.1",
}

// Coordinates contains coordinates for each area (in JP1-JP47 order)
var Coordinates = [][]float64{
	{43.064615, 141.346807}, // JP1 北海道
	{40.824308, 140.739998}, // JP2 青森
	{39.703619, 141.152684}, // JP3 岩手
	{38.268837, 140.8721},   // JP4 宮城
	{39.718614, 140.102364}, // JP5 秋田
	{38.240436, 140.363633}, // JP6 山形
	{37.750299, 140.467551}, // JP7 福島
	{36.341811, 140.446793}, // JP8 茨城
	{36.565725, 139.883565}, // JP9 栃木
	{36.390668, 139.060406}, // JP10 群馬
	{35.856999, 139.648849}, // JP11 埼玉
	{35.605057, 140.123306}, // JP12 千葉
	{35.689488, 139.691706}, // JP13 東京
	{35.447507, 139.642345}, // JP14 神奈川
	{37.902552, 139.023095}, // JP15 新潟
	{36.695291, 137.211338}, // JP16 富山
	{36.594682, 136.625573}, // JP17 石川
	{36.065178, 136.221527}, // JP18 福井
	{35.664158, 138.568449}, // JP19 山梨
	{36.651299, 138.180956}, // JP20 長野
	{35.391227, 136.722291}, // JP21 岐阜
	{34.97712, 138.383084},  // JP22 静岡
	{35.180188, 136.906565}, // JP23 愛知
	{34.730283, 136.508588}, // JP24 三重
	{35.004531, 135.86859},  // JP25 滋賀
	{35.021247, 135.755597}, // JP26 京都
	{34.686297, 135.519661}, // JP27 大阪
	{34.691269, 135.183071}, // JP28 兵庫
	{34.685334, 135.832742}, // JP29 奈良
	{34.225987, 135.167509}, // JP30 和歌山
	{35.503891, 134.237736}, // JP31 鳥取
	{35.472295, 133.0505},   // JP32 島根
	{34.661751, 133.934406}, // JP33 岡山
	{34.39656, 132.459622},  // JP34 広島
	{34.185956, 131.470649}, // JP35 山口
	{34.065718, 134.55936},  // JP36 徳島
	{34.340149, 134.043444}, // JP37 香川
	{33.841624, 132.765681}, // JP38 愛媛
	{33.559706, 133.531079}, // JP39 高知
	{33.606576, 130.418297}, // JP40 福岡
	{33.249442, 130.299794}, // JP41 佐賀
	{32.744839, 129.873756}, // JP42 長崎
	{32.789827, 130.741667}, // JP43 熊本
	{33.238172, 131.612619}, // JP44 大分
	{31.911096, 131.423893}, // JP45 宮崎
	{31.560146, 130.557978}, // JP46 鹿児島
	{26.2124, 127.680932},   // JP47 沖縄
}

// GenRandomDeviceInfo generates random device information
func GenRandomDeviceInfo() RandomDeviceInfo {
	// Randomly select Android version
	versions := make([]string, 0, len(VERSION_MAP))
	for v := range VERSION_MAP {
		versions = append(versions, v)
	}
	version := versions[rand.Intn(len(versions))]
	versionInfo := VERSION_MAP[version]

	// Randomly select build
	build := versionInfo.Builds[rand.Intn(len(versionInfo.Builds))]

	// Randomly select device model
	model := MODEL_LIST[rand.Intn(len(MODEL_LIST))]

	// Build device string: SDK.MODEL
	device := fmt.Sprintf("%s.%s", versionInfo.SDK, model)

	// Build User-Agent: Dalvik/2.1.0 (Linux; U; Android VERSION; MODEL/BUILD)
	userAgent := fmt.Sprintf("Dalvik/2.1.0 (Linux; U; Android %s; %s/%s)", version, model, build)

	// Randomly select app version
	appVersion := APP_VERSIONS[rand.Intn(len(APP_VERSIONS))]

	// Generate random user ID (32-character hexadecimal)
	userID := genRandomHexString(32)

	return RandomDeviceInfo{
		AppVersion: appVersion,
		UserID:     userID,
		UserAgent:  userAgent,
		Device:     device,
	}
}

// GenGPS generates GPS coordinates for the specified area (with random offset)
// areaID format: "JP1" - "JP47"
func GenGPS(areaID string) string {
	// Parse area number
	areaNum := parseAreaNumber(areaID)
	if areaNum < 1 || areaNum > 47 {
		// Default to Tokyo coordinates
		areaNum = 13
	}

	// Get base coordinates
	coords := Coordinates[areaNum-1]
	lat := coords[0]
	long := coords[1]

	// Add random offset (+/- 0 ~ 0.025 => 0 ~ 1.5' => +/- 0 ~ 2.77/2.13km)
	latOffset := rand.Float64() / 40.0
	if rand.Float64() > 0.5 {
		latOffset = -latOffset
	}
	longOffset := rand.Float64() / 40.0
	if rand.Float64() > 0.5 {
		longOffset = -longOffset
	}

	lat += latOffset
	long += longOffset

	return fmt.Sprintf("%.6f,%.6f,gps", lat, long)
}

// parseAreaNumber parses the area number from areaID
func parseAreaNumber(areaID string) int {
	// Remove "JP" prefix
	numStr := strings.TrimPrefix(areaID, "JP")
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return -1
	}
	return num
}

// NewRandomDeviceInfo creates a RandomDeviceInfo with custom parameters
// appVersion: app version, e.g., "7.4.10"
// userID: user ID, 32-character hexadecimal string
// userAgent: User-Agent string
// device: device identifier, e.g., "29.SM-N950N"
func NewRandomDeviceInfo(appVersion, userID, userAgent, device string) RandomDeviceInfo {
	return RandomDeviceInfo{
		AppVersion: appVersion,
		UserID:     userID,
		UserAgent:  userAgent,
		Device:     device,
	}
}

// GenLsid generates a random 32-character hexadecimal lsid for stream URLs
func GenLsid() string {
	return genRandomHexString(32)
}

// genRandomHexString generates a random hexadecimal string of specified length
func genRandomHexString(length int) string {
	const hex = "0123456789abcdef"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = hex[rand.Intn(len(hex))]
	}
	return string(result)
}
