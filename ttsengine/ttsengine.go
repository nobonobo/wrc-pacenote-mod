package ttsengine

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aethiopicuschan/nanoda"
	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/nobonobo/wrc-pacenote-mod/config"
)

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
	coreDir := filepath.Join(config.Config.Root, "voicevox_core")
	v, err := nanoda.NewVoicevox(
		filepath.Join(coreDir, "voicevox_core.dll"),
		filepath.Join(coreDir, "open_jtalk_dic_utf_8-1.11"),
		filepath.Join(coreDir, "model"))
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
