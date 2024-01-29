package config

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

var Config = struct {
	Listen      string `json:"listen"`
	Forward     string `json:"forward"`
	WebListen   string `json:"web-listen"`
	LogDir      string `json:"log-dir"`
	VoiceVoxDir string `json:"voicevox-dir"`
}{
	Listen:    "127.0.0.1:20777",
	Forward:   "",
	WebListen: "127.0.0.1:8080",
	LogDir:    os.ExpandEnv(filepath.Join("${USERPROFILE}", "Documents", "My Games", "WRC", "pacenotes")),
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	d, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	Config.VoiceVoxDir = filepath.Join(d, "voicevox_core")
	flag.StringVar(&Config.Listen, "listen", Config.Listen, "listen address")
	flag.StringVar(&Config.Forward, "forward", Config.Forward, "forward address")
	flag.StringVar(&Config.WebListen, "web-listen", Config.WebListen, "web listen address")
	flag.StringVar(&Config.LogDir, "log-dir", Config.LogDir, "log directory")
	flag.StringVar(&Config.VoiceVoxDir, "voicevox-dir", Config.VoiceVoxDir, "voicevox directory")
	flag.Parse()
}
