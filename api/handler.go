package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	svg "github.com/ajstarks/svgo"

	"github.com/nobonobo/wrc-pacenote-mod/config"
	"github.com/nobonobo/wrc-pacenote-mod/easportswrc"
)

type Result struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Region struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Content string  `json:"content"`
}

type Regions []Region

func hello(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Write([]byte("Hello World!\n"))
}

func GetStageByPath(dir string) *easportswrc.Stage {
	l, s := path.Split(path.Clean(dir))
	loc, err := strconv.Atoi(path.Base(l))
	if err != nil {
		return nil
	}
	locIdx := loc - 1
	if locIdx < 0 || locIdx >= len(easportswrc.Locations) {
		return nil
	}
	location := easportswrc.Locations[locIdx]
	ss, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	ssIdx := ss - 1
	if ssIdx < 0 || ssIdx >= len(location.Stages) {
		return nil
	}
	stage := location.Stages[ssIdx]
	return &easportswrc.Stage{
		ID: easportswrc.StageID{
			Location: loc,
			Stage:    ss,
		},
		Location: location.Name,
		Stage:    stage,
	}
}

func GetFilePathFromStage(stage *easportswrc.Stage) string {
	return filepath.Join(
		fmt.Sprintf("%02d.%s", stage.ID.Location, stage.Location),
		fmt.Sprintf("%02d.%s", stage.ID.Stage, stage.Stage),
	)
}

func GetFilePath(p string) string {
	dir, name := path.Split(p)
	stage := GetStageByPath(dir)
	if stage == nil {
		return ""
	}
	return filepath.Join(
		GetFilePathFromStage(stage),
		name,
	)
}

func locations(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	locations := []map[string]interface{}{}
	for loc, location := range easportswrc.Locations {
		stages := []easportswrc.Stage{}
		for ss, name := range location.Stages {
			stage := easportswrc.Stage{
				ID: easportswrc.StageID{
					Location: loc + 1,
					Stage:    ss + 1,
				},
				Location: location.Name,
				Stage:    name,
			}
			dir := filepath.Join(config.Config.LogDir, GetFilePathFromStage(&stage))
			if _, err := os.Stat(filepath.Join(dir, "capture.wav")); err != nil {
				if os.IsNotExist(err) {
					continue
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "telemetry.log")); err != nil {
				if os.IsNotExist(err) {
					continue
				}
			}
			stages = append(stages, stage)
		}
		if len(stages) == 0 {
			continue
		}
		locations = append(locations, map[string]interface{}{
			"Name":   location.Name,
			"Stages": stages,
		})
	}
	if err := json.NewEncoder(w).Encode(locations); err != nil {
		log.Println("locations encode failed:", err)
		b, _ := json.Marshal(Result{false, err.Error()})
		http.Error(w, string(b), http.StatusInternalServerError)
	}
}

func stageName(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	stage := GetFilePath(r.URL.Path)
	if err := json.NewEncoder(w).Encode(strings.Replace(stage, "\\", " / ", -1)); err != nil {
		errMsg := fmt.Sprintf("stage name get failed: %s", err.Error())
		log.Println(errMsg)
		b, _ := json.Marshal(Result{false, errMsg})
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}
}

func files(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	stage := GetFilePath(r.URL.Path)
	if stage == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fpath := filepath.Join(config.Config.LogDir, stage)
	log.Printf("file serve: %q", fpath)
	http.ServeFile(w, r, fpath)
}

func getRegions(w io.Writer, r *http.Request) error {
	stage := GetFilePath(r.URL.Path)
	if stage == "" {
		return fmt.Errorf("stage not found: %q", r.URL.Path)
	}
	fpath := filepath.Join(config.Config.LogDir, stage, "regions.log")
	log.Println("regions load from:", fpath)
	regions := Regions{}
	fp, err := os.Open(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			json.NewEncoder(w).Encode(regions)
			return nil
		}
		return err
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		text := scanner.Text()
		fields := strings.Split(text, ",")
		if len(fields) != 3 {
			continue
		}
		start, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			continue
		}
		end, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		regions = append(regions, Region{
			Start:   start,
			End:     end,
			Content: fields[2],
		})
	}
	if err := json.NewEncoder(w).Encode(regions); err != nil {
		return err
	}
	return nil
}

func postRegions(w http.ResponseWriter, r *http.Request) error {
	regions := Regions{}
	if err := json.NewDecoder(r.Body).Decode(&regions); err != nil {
		return err
	}
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Start < regions[j].Start
	})
	stage := GetFilePath(r.URL.Path)
	if err := func() error {
		fpath := filepath.Join(config.Config.LogDir, stage, "regions.log")
		log.Println("regions save to:", fpath)
		fp, err := os.Create(fpath)
		if err != nil {
			return fmt.Errorf("region.log create failed: %w", err)
		}
		defer fp.Close()
		for _, region := range regions {
			if _, err := fmt.Fprintf(fp, "%f,%f,%s\n", region.Start, region.End, region.Content); err != nil {
				return err
			}
		}
		if err := fp.Sync(); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		return fmt.Errorf("region.log save failed: %w", err)
	}
	// generate pacenote
	fpath := filepath.Join(config.Config.LogDir, stage, "pacenote.log")
	output, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("pacenote.log create failed: %w", err)
	}
	defer output.Close()
	fp, err := os.Open(filepath.Join(config.Config.LogDir, stage, "telemetry.log"))
	if err != nil {
		return err
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	index := 0
	for scanner.Scan() {
		text := scanner.Text()
		fields := strings.Split(text, ",")
		if len(fields) != 5 {
			continue
		}
		ts, err := strconv.Atoi(fields[1]) // duration
		if err != nil {
			return err
		}
		pos := []float64{}
		for i := 0; i < 3; i++ {
			p, err := strconv.ParseFloat(fields[i+2], 64)
			if err != nil {
				return err
			}
			pos = append(pos, p)
		}
		if index >= len(regions) {
			break
		}
		region := regions[index]
		if region.Start < float64(ts)/float64(time.Second) {
			fmt.Fprintf(output, "%f,%f,%f,%s\n", pos[0], pos[1], pos[2], region.Content)
			index++
		}
	}
	if err := output.Sync(); err != nil {
		return fmt.Errorf("pacenote.log save failed: %w", err)
	}
	if err := json.NewEncoder(w).Encode(Result{true, ""}); err != nil {
		return err
	}
	return nil
}

func regions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	default:
		errMsg := http.StatusText(http.StatusMethodNotAllowed)
		log.Println(errMsg)
		b, _ := json.Marshal(Result{false, errMsg})
		http.Error(w, string(b), http.StatusMethodNotAllowed)
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "GET":
		if err := getRegions(w, r); err != nil {
			log.Println(err)
			b, _ := json.Marshal(Result{false, err.Error()})
			http.Error(w, string(b), http.StatusBadRequest)
		}
		return
	case "POST":
		if err := postRegions(w, r); err != nil {
			log.Println(err)
			b, _ := json.Marshal(Result{false, err.Error()})
			http.Error(w, string(b), http.StatusBadRequest)
		}
	}
}

func mapgen(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	stage := GetFilePath(r.URL.Path)
	fpath := filepath.Join(config.Config.LogDir, stage, "telemetry.log")
	fp, err := os.Open(fpath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer fp.Close()
	log.Println("mapgen serve from:", fpath)
	listT := []string{}
	listZ := []int{}
	listX := []int{}
	maxZ := -10000000
	minZ := 10000000
	maxX := -10000000
	minX := 10000000
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		text := scanner.Text()
		fields := strings.Split(text, ",")
		if len(fields) != 5 {
			continue
		}
		listT = append(listT, fields[1])
		pos := []float64{}
		for i := 2; i < 5; i++ {
			p, err := strconv.ParseFloat(fields[i], 64)
			if err != nil {
				log.Print(err)
				continue
			}
			pos = append(pos, p*10)
		}
		z, x := int(pos[2]), int(pos[0])
		if z > maxZ {
			maxZ = z
		}
		if z < minZ {
			minZ = z
		}
		if x > maxX {
			maxX = x
		}
		if x < minX {
			minX = x
		}
		listZ = append(listZ, z)
		listX = append(listX, x)
	}
	canvas := svg.New(w)
	canvas.StartviewUnit(256, 256, "px", minX-1000, minZ-1000, maxX-minX+2000, maxZ-minZ+2000)
	canvas.Style("",
		"line{stroke:cyan;stroke-width:10vh}",
		"circle{fill:red;stroke:black;stroke-width:10vh}",
		"text{text-anchor:middle;font-size:200vh;fill:silver}",
	)
	canvas.Text((maxX-minX)/2+minX, maxZ+800, strings.Replace(stage, "\\", " / ", -1))
	canvas.Gid("points")
	const NA = -99999999
	lastZ, lastX := NA, NA
	for i := 0; i < len(listT); i++ {
		if lastZ != NA && lastX != NA {
			canvas.Gid(listT[i])
			canvas.Line(lastX, lastZ, listX[i], listZ[i])
			canvas.Gend()
		}
		lastZ, lastX = listZ[i], listX[i]
	}
	canvas.Gend()
	canvas.Gid("vehicle")
	canvas.Circle(listX[0], listZ[0], 100)
	canvas.Gend()
	canvas.End()
}

func Setup(ctx context.Context) {
	mux := http.NewServeMux()
	http.Handle("/api/", http.StripPrefix("/api", mux))
	mux.Handle("/hello", http.HandlerFunc(hello))
	mux.Handle("/locations", http.HandlerFunc(locations))
	mux.Handle("/stage/", http.StripPrefix("/stage", http.HandlerFunc(stageName)))
	mux.Handle("/files/", http.StripPrefix("/files", http.HandlerFunc(files)))
	mux.Handle("/regions/", http.StripPrefix("/regions", http.HandlerFunc(regions)))
	mux.Handle("/map/", http.StripPrefix("/map", http.HandlerFunc(mapgen)))
}
