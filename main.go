package main

import (
	"fmt"
	"os/exec"
	"radikojp/hook"
)

func main() {

	token := "X-Radiko-AuthToken:" + hook.Auth()
	cmd := exec.Command("ffplay", "-i", "https://c-radiko.smartstream.ne.jp/QRR/_definst_/simul-stream.stream/playlist.m3u8?station_id=QRR&l=30&lsid=5e586af5ccb3b0b2498abfb19eaa8472&type=b", "-headers", token)
	//cmd := exec.Command("ffplay", "-i", "https://c-radiko.smartstream.ne.jp/QRR/_definst_/simul-stream.stream/playlist.m3u8?station_id=QRR&l=30&lsid=5e586af5ccb3b0b2498abfb19eaa8472&type=b")
	err2 := cmd.Run()
	if err2 != nil {
		fmt.Println(err2)
	}

}
