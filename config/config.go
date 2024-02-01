package config

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

var Config = struct {
	Listen      string `json:"listen"`
	Forward     string `json:"forward"`
	WebListen   string `json:"web-listen"`
	LogDir      string `json:"log-dir"`
	VoiceVoxDir string
	Root        string
	Documents   string
}{
	Listen:    "127.0.0.1:20777",
	Forward:   "",
	WebListen: "127.0.0.1:8080",
	LogDir:    "",
	Root:      ".",
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	Config.Root = getRootDir()

	doc, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
	if err != nil {
		log.Fatal(err)
	}
	Config.Documents = doc
	WRCDocumentRoot := os.ExpandEnv(filepath.Join(Config.Documents, "My Games", "WRC"))
	Config.LogDir = filepath.Join(WRCDocumentRoot, "pacenotes")
	Config.VoiceVoxDir = filepath.Join(WRCDocumentRoot, "voicevox_core")
	if err := windows.SetDllDirectory(Config.VoiceVoxDir); err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&Config.Listen, "listen", Config.Listen, "listen address")
	flag.StringVar(&Config.Forward, "forward", Config.Forward, "forward address")
	flag.StringVar(&Config.WebListen, "web-listen", Config.WebListen, "web listen address")
	flag.StringVar(&Config.LogDir, "log-dir", Config.LogDir, "log directory")
	flag.Parse()
}
