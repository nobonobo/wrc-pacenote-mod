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
	qDicts, err := Init(s, Dict)
	if err != nil {
		return err
	}
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
				qm, ok := qDicts[v]
				if !ok {
					q, err := makeAudioQuery(s, v)
					if err != nil {
						return err
					}
					qm = q
				}
				if err := playback(ctxOto, s, qm); err != nil {
					return err
				}
			}
		}
	}
}
