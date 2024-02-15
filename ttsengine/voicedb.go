package ttsengine

import (
	_ "embed"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/aethiopicuschan/nanoda"
	"github.com/nobonobo/wrc-pacenote-mod/config"
)

var (
	ActorID           = 3
	Pitch             = 0.0
	Intnation         = 1.0
	Speed             = 1.4
	Volume            = 1.8
	Pause             = 0.1
	PrePhonemeLength  = 0.0
	PostPhonemeLength = 0.0
)

func init() {
	flag.IntVar(&ActorID, "actor", ActorID, "actor id")
	flag.Float64Var(&Pitch, "pitch", Pitch, "pitch")
	flag.Float64Var(&Intnation, "intnation", Volume, "intnation")
	flag.Float64Var(&Speed, "speed", Speed, "base speed")
	flag.Float64Var(&Volume, "volume", Volume, "volume magnification")
	flag.Float64Var(&Pause, "pause", Pause, "pause magnification")
	flag.Float64Var(&PrePhonemeLength, "pre-phoneme", PrePhonemeLength, "pre-phoneme-length")
	flag.Float64Var(&PostPhonemeLength, "post-phoneme", PostPhonemeLength, "post-phoneme-length")
}

type AQ struct {
	Text      string  `json:"text"`
	Speed     float64 `json:"speed"`
	Pitch     float64 `json:"pitch"`
	Intnation float64 `json:"intnation"`
	Volume    float64 `json:"volume"`
}

type AudioDict map[string]nanoda.AudioQuery

var (
	Dict      map[string]AQ
	stageDict = NewDict()
)

//go:embed base.json
var base []byte

func NewDict() AudioDict {
	return map[string]nanoda.AudioQuery{}
}

func (d AudioDict) Add(s string) {
	if _, ok := d[s]; ok {
		return
	}
	q, err := makeAudioQuery(synthesizer, s)
	if err != nil {
		log.Println(err)
		return
	}
	d[s] = q
}

func SetDict(d AudioDict) {
	stageDict = d
}

func writeDictionary(dstName string) error {
	os.MkdirAll(filepath.Dir(dstName), 0777)
	dst, err := os.Create(dstName) // コピー先ファイルを作成する
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := dst.Write(base); err != nil {
		return err
	}
	if err := dst.Sync(); err != nil {
		return err
	}
	return nil
}

func init() {
	fpath := filepath.Join(config.Config.LogDir, "dictionary.json")
	if _, err := os.Stat(fpath); err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
		if err := writeDictionary(fpath); err != nil {
			log.Fatal(err)
		}
	}
	log.Println("loading dictionary.json")
	defer log.Println("dictionary.json loading completed")
	fp, err := os.Open(fpath)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	if err := json.NewDecoder(fp).Decode(&Dict); err != nil {
		log.Fatal(err)
	}
}

func makeAudioQuery(s nanoda.Synthesizer, text string) (nanoda.AudioQuery, error) {
	q, err := s.CreateAudioQuery(text, nanoda.StyleId(ActorID))
	if err != nil {
		return nanoda.AudioQuery{}, err
	}
	q.IntonationScale = Intnation
	q.PitchScale = Pitch
	q.SpeedScale = Speed
	q.VolumeScale = Volume
	q.PrePhonemeLength = PrePhonemeLength
	q.PostPhonemeLength = PostPhonemeLength
	for _, p := range q.AccentPhrases[1:] {
		if p.PauseMora != nil {
			p.PauseMora.VowelLength *= Pause
		}
	}
	return q, nil
}

func Init(s nanoda.Synthesizer, dict map[string]AQ) (map[string]nanoda.AudioQuery, error) {
	res := map[string]nanoda.AudioQuery{}
	for k, v := range dict {
		q, err := makeAudioQuery(s, v.Text)
		if err != nil {
			return nil, err
		}
		if v.Intnation != 0.0 {
			q.IntonationScale = v.Intnation
		}
		if v.Pitch != 0.0 {
			q.PitchScale = v.Pitch
		}
		if v.Speed != 0.0 {
			q.SpeedScale *= v.Speed
		}
		if v.Volume != 0.0 {
			q.VolumeScale *= v.Volume
		}
		res[k] = q
	}
	return res, nil
}
