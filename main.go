package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deltammo/octree"
	"github.com/ebitengine/oto/v3"
	"github.com/moutend/go-wav"

	"github.com/nobonobo/wrc-pacenote-mod/api"
	"github.com/nobonobo/wrc-pacenote-mod/capture"
	"github.com/nobonobo/wrc-pacenote-mod/config"
	"github.com/nobonobo/wrc-pacenote-mod/easportswrc"
	"github.com/nobonobo/wrc-pacenote-mod/ttsengine"
)

func getLogDir(stageLength float64) string {
	stage := easportswrc.GetStage(stageLength)
	if stage == nil {
		return filepath.Join(config.Config.LogDir, fmt.Sprintf("%f", stageLength))
	}
	return filepath.Join(config.Config.LogDir,
		fmt.Sprintf("%02d.%s", stage.ID.Location, stage.Location),
		fmt.Sprintf("%02d.%s", stage.ID.Stage, stage.Stage),
	)
}

func isChange(prev, next *easportswrc.PacketEASportsWRC) bool {
	if prev == nil {
		return true
	}
	return prev.StageCurrentDistance != next.StageCurrentDistance ||
		prev.VehiclePositionX != next.VehiclePositionX ||
		prev.VehiclePositionY != next.VehiclePositionY ||
		prev.VehiclePositionZ != next.VehiclePositionZ
}

func uniqueRename(fpath string) string {
	next := fpath
	idx := 0
	for {
		idx++
		if _, err := os.Stat(next); err != nil {
			if os.IsNotExist(err) {
				return next
			}
		}
		next = fpath + fmt.Sprintf(".%d", idx)
	}
}

func logging() func(context.Context, *easportswrc.PacketEASportsWRC) error {
	currentDuration := uint64(0)
	setCurrent := func(v time.Duration) {
		atomic.StoreUint64(&currentDuration, uint64(v))
	}
	getCurrent := func() time.Duration {
		return time.Duration(atomic.LoadUint64(&currentDuration))
	}
	finished := uint32(0)
	setFinished := func(b bool) {
		v := uint32(0)
		if b {
			v = 1
		}
		atomic.StoreUint32(&finished, v)
	}
	getFinished := func() bool {
		return atomic.LoadUint32(&finished) == 1
	}
	closer := func() {}
	closeFuncs := []func(){}
	logDir := ""
	logFile := (*bytes.Buffer)(nil)
	wavFile := (*wav.File)(nil)
	lastDistance := float64(100000)
	timeout := (*time.Timer)(nil)
	lastPacket := (*easportswrc.PacketEASportsWRC)(nil)
	var finishCnt = 0
	isFinished := func(pkt *easportswrc.PacketEASportsWRC) bool {
		if pkt == nil {
			return false
		}
		finish := pkt.StageCurrentDistance > pkt.StageLength-1000 && pkt.VehicleClutch == 1.0 && pkt.VehicleBrake == 1.0
		if finish {
			finishCnt++
		}
		return finishCnt > 3
	}
	return func(ctx context.Context, pkt *easportswrc.PacketEASportsWRC) error {
		defer func() {
			lastPacket = pkt
			lastDistance = pkt.StageCurrentDistance
			setFinished(isFinished(lastPacket))
		}()
		if pkt.PacketUid%500 == 0 {
			log.Printf("packet: %v", pkt)
		}
		if lastDistance != 0.0 && pkt.StageCurrentDistance == 0 {
			finishCnt = 0
			setFinished(false)
			if closer != nil {
				closer()
			}
			closer = sync.OnceFunc(func() {
				for _, f := range closeFuncs {
					f()
				}
				closeFuncs = nil
			})
			logDir = getLogDir(pkt.StageLength)
			logName := filepath.Join(logDir, "telemetry.log")
			os.MkdirAll(logDir, 0755)
			logName = uniqueRename(logName)
			log.Printf("logger (re)start: %q", logName)
			setCurrent(0)
			logFile = bytes.NewBuffer(nil)
			closeFuncs = append(closeFuncs, func() {
				if getFinished() {
					os.WriteFile(logName, logFile.Bytes(), 0o644)
					log.Printf("log saved: %q", logName)
				} else {
					log.Print("log save skiiped")
				}
				logFile = nil
			})
			ctx, cancel := context.WithCancel(ctx)
			closeFuncs = append(closeFuncs, cancel)
			timeout = time.AfterFunc(3*time.Second, closer)
			go func(ctx context.Context) {
				log.Println("audio recorder: started")
				defer log.Println("audio recorder: terminated")
				defer closer()
				output := func(v capture.Chunk) {
					if !getFinished() {
						timeout.Reset(3 * time.Second)
					}
					setCurrent(v.CurrentDuration)
					if wavFile == nil {
						w, err := wav.New(
							int(v.Format.SamplesPerSec),
							int(v.Format.BitsPerSample),
							int(v.Format.Channels),
						)
						if err != nil {
							log.Println(err)
							return
						}
						wavFile = w
						closeFuncs = append(closeFuncs, func() {
							b, err := wav.Marshal(wavFile)
							if err != nil {
								log.Println(err)
								return
							}
							wavName := filepath.Join(logDir, "capture.wav")
							wavName = uniqueRename(wavName)
							if getFinished() {
								if err := os.WriteFile(wavName, b, 0644); err != nil {
									log.Println(err)
									return
								}
								log.Printf("wav saved: %q", wavName)
							} else {
								log.Println("wav save skipped")
							}
							wavFile = nil
						})
					}
					if _, err := wavFile.Write(v.Buffer); err != nil {
						log.Println(err)
						return
					}
				}
				if err := capture.Capture(ctx, output); err != nil {
					log.Println(err)
					return
				}
			}(ctx)
		}
		if logFile != nil && isChange(lastPacket, pkt) {
			fmt.Fprintf(logFile, "%d,%d,%f,%f,%f\n",
				pkt.PacketUid,
				getCurrent(),
				pkt.VehiclePositionX,
				pkt.VehiclePositionY,
				pkt.VehiclePositionZ,
			)
		}
		return nil
	}
}

type Pacenote struct {
	ID      int     `json:"id"`
	Message string  `json:"message"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
}

func normal(speechCh chan<- string) func(context.Context, *easportswrc.PacketEASportsWRC) error {
	const (
		size = 100000
		half = size / 2
	)
	sparseDB := (*octree.Octree)(nil)
	lastIndex := -1
	lastDistance := 0.0
	lastStageLength := -1.0
	pacenoteInvalid := false
	cnt := 0
	return func(ctx context.Context, pkt *easportswrc.PacketEASportsWRC) error {
		if lastDistance != 0 && pkt.StageCurrentDistance == 0 {
			log.Println("reload pacenote")
			lastDistance = 0.0
			lastStageLength = -1.0
			pacenoteInvalid = false
		}
		if lastStageLength != pkt.StageLength {
			sparseDB = nil
			lastIndex = -1
			lastStageLength = pkt.StageLength
		}
		if sparseDB == nil && !pacenoteInvalid {
			dir := getLogDir(pkt.StageLength)
			fpath := filepath.Join(dir, "pacenote.log")
			fp, err := os.Open(fpath)
			if err != nil {
				pacenoteInvalid = true
				return err
			}
			log.Printf("pacenote loading start: %q", fpath)
			tree, err := octree.New(0, 0, 0, size, 8, 0)
			if err != nil {
				pacenoteInvalid = true
				return err
			}
			scanner := bufio.NewScanner(fp)
			index := 1
			for scanner.Scan() {
				fields := strings.Split(scanner.Text(), ",")
				message := strings.Join(fields[3:], " ")
				x, err := strconv.ParseFloat(fields[0], 64)
				if err != nil {
					log.Println(err)
					continue
				}
				y, err := strconv.ParseFloat(fields[1], 64)
				if err != nil {
					log.Println(err)
					continue
				}
				z, err := strconv.ParseFloat(fields[2], 64)
				if err != nil {
					log.Println(err)
					continue
				}
				if err := tree.Set(&Pacenote{
					ID: index, Message: message,
					X: x, Y: y, Z: z,
				}, x+half, y+half, z+half); err != nil {
					log.Println(err)
					continue
				}
				index++
			}
			fp.Close()
			sparseDB = tree
			log.Println("pacenote loading completed")
		}
		if pkt.StageCurrentDistance == 0 {
			return nil
		}
		cnt++
		x := float64(pkt.VehiclePositionX)
		y := float64(pkt.VehiclePositionY)
		z := float64(pkt.VehiclePositionZ)
		lastDistance = pkt.StageCurrentDistance
		searchDist := 100.0
		if lastIndex < 0 {
			searchDist = 10
		}
		find := sparseDB.FindSlice(x+half, y+half, z+half, searchDist) // dist under 100m
		if len(find) == 0 {
			return nil
		}
		records := make([]*Pacenote, 0, len(find))
		for _, v := range find {
			records = append(records, v.(*Pacenote))
		}
		sort.Slice(records, func(i, j int) bool {
			return records[i].ID < records[j].ID
		})
		for {
			if records[0].ID > lastIndex {
				break
			}
			records = records[1:]
			if len(records) == 0 {
				return nil
			}
		}
		r := records[0]
		dx, dy, dz := x-r.X, y-r.Y, z-r.Z
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if dist < 10 {
			log.Println("speech:", r.Message)
			speechCh <- r.Message
			lastIndex = r.ID
		}
		return nil
	}
}

func receiver(speechCh chan<- string) func(ctx context.Context) {
	var lastDistance = 0.0
	return func(ctx context.Context) {
		var dest *net.UDPAddr
		if config.Config.Forward != "" {
			addr, err := net.ResolveUDPAddr("udp", config.Config.Forward)
			if err != nil {
				log.Fatal(err)
			}
			dest = addr
		}
		conn, err := net.ListenPacket("udp", config.Config.Listen)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("udp listening start:", config.Config.Listen)
		defer log.Println("udp listener terminated:", config.Config.Listen)
		go func() {
			<-ctx.Done()
			conn.Close()
		}()
		recording := logging()
		playback := normal(speechCh)
		recodingMode := false
		buf := make([]byte, 4096)
		for {
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				log.Print(err)
				return
			}
			if dest != nil {
				if _, err := conn.WriteTo(buf[:n], dest); err != nil {
					log.Print(err)
				}
			}
			if n != easportswrc.PacketEASportsWRCLength {
				continue
			}
			pkt := new(easportswrc.PacketEASportsWRC)
			if err := pkt.UnmarshalBinary(buf[:n]); err != nil {
				log.Print(err)
				continue
			}
			if lastDistance != pkt.StageLength {
				recodingMode = false
				lastDistance = pkt.StageLength
				dir := getLogDir(pkt.StageLength)
				fpath := filepath.Join(dir, "pacenote.log")
				if _, err := os.Stat(fpath); err != nil {
					if os.IsNotExist(err) {
						log.Printf("pacenotes.log not found: %q", fpath)
						recodingMode = true
					}
				}
				if recodingMode {
					speechCh <- "recording-mode"
				}
			}
			if recodingMode {
				if err := recording(ctx, pkt); err != nil {
					log.Print(err)
				}
			} else {
				if err := playback(ctx, pkt); err != nil {
					log.Print(err)
				}
			}
		}
	}
}

func urlLog(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	})
}

func serve(ctx context.Context) error {
	setup(ctx)
	l, err := net.Listen("tcp", config.Config.WebListen)
	if err != nil {
		return err
	}
	defer l.Close()
	server := &http.Server{Handler: urlLog(http.DefaultServeMux)}
	api.Setup(ctx)
	log.Println("http listening start:", config.Config.WebListen)
	defer log.Println("http listener terminated:", config.Config.WebListen)
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()
	if err := server.Serve(l); err != nil {
		return err
	}
	return nil
}

func main() {
	runtime.LockOSThread()
	var wg sync.WaitGroup
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())

	ctxOto, _, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   48000,
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		log.Fatal(err)
	}

	speechCh := make(chan string, 10)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-signalChan
		cancel()
		close(speechCh)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := serve(ctx); err != nil {
			log.Print(err)
		}
	}()

	go receiver(speechCh)(ctx)

	for {
		if err := ttsengine.StartEngine(ctx, ctxOto, speechCh); err != nil {
			log.Print(err)
		}
		select {
		default:
			continue
		case <-ctx.Done():
		}
		break
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
}
