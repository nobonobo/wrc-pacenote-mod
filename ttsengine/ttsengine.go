package ttsengine

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aethiopicuschan/nanoda"
	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/nobonobo/wrc-pacenote-mod/config"
)

const (
	downloadUrl = "https://github.com/VOICEVOX/voicevox_core/releases/download/0.15.0-preview.13/download-windows-x64.exe"
)

func download(u, folder string) error {
	info, err := url.Parse(u)
	if err != nil {
		return err
	}
	fname := filepath.Join(folder, filepath.Base(info.Path))
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func isInstalled(folder string) bool {
	files := []string{
		"voicevox_core.dll",
		"onnxruntime_providers_shared.dll",
		"onnxruntime.dll",
		"open_jtalk_dic_utf_8-1.11",
		"model",
	}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(folder, f)); err != nil {
			return false
		}
	}
	return true
}

func installVoiceVox(folder string) error {
	if err := download(downloadUrl, folder); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "./download-windows-x64.exe",
		"--device", "cpu", "--version", "0.15.0-preview.13",
	)
	cmd.Dir = folder
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Println("install voicevox_core:", folder)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func init() {
	folder := config.Config.VoiceVoxDir
	if isInstalled(folder) {
		return
	}
	if err := os.RemoveAll(folder); err != nil {
		log.Fatal(err)
	}
	if err := installVoiceVox(filepath.Dir(folder)); err != nil {
		log.Fatal(err)
	}
}

func playback(ctxOto *oto.Context, s nanoda.Synthesizer, q nanoda.AudioQuery) error {
	w, err := s.Synthesis(q, nanoda.StyleId(ActorID))
	if err != nil {
		return err
	}
	defer w.Close()
	decoded, err := wav.DecodeWithoutResampling(w)
	if err != nil {
		return err
	}
	p := ctxOto.NewPlayer(decoded)
	defer p.Close()
	p.Play()
	for p.IsPlaying() {
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

var dictionary = map[string]nanoda.AudioQuery{}

func StartEngine(ctx context.Context, ctxOto *oto.Context, in <-chan string) error {
	v, err := nanoda.NewVoicevox(
		filepath.Join(config.Config.VoiceVoxDir, "voicevox_core.dll"),
		filepath.Join(config.Config.VoiceVoxDir, "open_jtalk_dic_utf_8-1.11"),
		filepath.Join(config.Config.VoiceVoxDir, "model"))
	if err != nil {
		return err
	}
	s, err := v.NewSynthesizer()
	if err != nil {
		return err
	}
	if err := s.LoadModelsFromStyleId(nanoda.StyleId(ActorID)); err != nil {
		return err
	}
	d, err := Init(s, Dict)
	if err != nil {
		return err
	}
	dictionary = d
	log.Println("TTS Engine started")
	defer log.Println("TTS Engine stopped")
	for {
		select {
		case <-ctx.Done():
			return nil
		case words := <-in:
			for _, v := range strings.Fields(words) {
				if v == "unknown" {
					continue
				}
				qm, ok := dictionary[v]
				if !ok {
					q, err := makeAudioQuery(s, v)
					if err != nil {
						return err
					}
					dictionary[v] = q
					qm = q
				}
				if err := playback(ctxOto, s, qm); err != nil {
					return err
				}
			}
		}
	}
}
