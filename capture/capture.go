package capture

import (
	"context"
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

type WavFormat struct {
	Channels      uint16
	SamplesPerSec uint32
	BitsPerSample uint16
}

type Chunk struct {
	Format          *WavFormat
	CurrentDuration time.Duration
	Buffer          []byte
}

func Capture(ctx context.Context, output func(Chunk)) error {
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return err
	}
	defer ole.CoUninitialize()

	var mmdCapturee *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmdCapturee); err != nil {
		return err
	}
	defer mmdCapturee.Release()

	var mmdCapture *wca.IMMDevice
	if err := mmdCapturee.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmdCapture); err != nil {
		return err
	}
	defer mmdCapture.Release()

	var mmdRender *wca.IMMDevice
	if err := mmdCapturee.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmdRender); err != nil {
		return err
	}
	defer mmdRender.Release()

	var ps *wca.IPropertyStore
	if err := mmdCapture.OpenPropertyStore(wca.STGM_READ, &ps); err != nil {
		return err
	}
	defer ps.Release()

	var pv wca.PROPVARIANT
	if err := ps.GetValue(&wca.PKEY_Device_FriendlyName, &pv); err != nil {
		return err
	}
	log.Printf("Capturing audio from: %s\n", pv.String())

	var cac *wca.IAudioClient
	if err := mmdCapture.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &cac); err != nil {
		return err
	}
	defer cac.Release()

	var rac *wca.IAudioClient
	if err := mmdRender.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &rac); err != nil {
		return err
	}
	defer rac.Release()

	var wfx *wca.WAVEFORMATEX
	if err := rac.GetMixFormat(&wfx); err != nil {
		return err
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(wfx)))

	wfx.WFormatTag = 1
	wfx.NBlockAlign = (wfx.WBitsPerSample / 8) * wfx.NChannels
	wfx.NAvgBytesPerSec = wfx.NSamplesPerSec * uint32(wfx.NBlockAlign)
	wfx.CbSize = 0

	format := &WavFormat{
		SamplesPerSec: wfx.NSamplesPerSec,
		Channels:      wfx.NChannels,
		BitsPerSample: wfx.WBitsPerSample,
	}

	log.Println("--------")
	log.Printf("Format: PCM %d bit signed integer\n", wfx.WBitsPerSample)
	log.Printf("Rate: %d Hz\n", wfx.NSamplesPerSec)
	log.Printf("Channels: %d\n", wfx.NChannels)
	log.Println("--------")

	var defaultPeriod wca.REFERENCE_TIME
	var minimumPeriod wca.REFERENCE_TIME
	var latency time.Duration
	if err := rac.GetDevicePeriod(&defaultPeriod, &minimumPeriod); err != nil {
		return err
	}
	latency = time.Duration(int(defaultPeriod) * 100)

	log.Println("Default period: ", defaultPeriod)
	log.Println("Minimum period: ", minimumPeriod)
	log.Println("Latency: ", latency)

	if err := cac.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, wca.AUDCLNT_STREAMFLAGS_EVENTCALLBACK|wca.AUDCLNT_STREAMFLAGS_LOOPBACK, defaultPeriod, 0, wfx, nil); err != nil {
		return err
	}
	if err := rac.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, wca.AUDCLNT_STREAMFLAGS_EVENTCALLBACK, defaultPeriod, 0, wfx, nil); err != nil {
		return err
	}

	fakeAudioReadyEvent := wca.CreateEventExA(0, 0, 0, wca.EVENT_MODIFY_STATE|wca.SYNCHRONIZE)
	defer wca.CloseHandle(fakeAudioReadyEvent)

	if err := cac.SetEventHandle(fakeAudioReadyEvent); err != nil {
		return err
	}

	audioReadyEvent := wca.CreateEventExA(0, 0, 0, wca.EVENT_MODIFY_STATE|wca.SYNCHRONIZE)
	defer wca.CloseHandle(audioReadyEvent)

	if err := rac.SetEventHandle(audioReadyEvent); err != nil {
		return err
	}

	var bufferFrameSizeRender uint32
	if err := rac.GetBufferSize(&bufferFrameSizeRender); err != nil {
		return err
	}

	var bufferFrameSize uint32
	if err := cac.GetBufferSize(&bufferFrameSize); err != nil {
		return err
	}

	log.Printf("Allocated buffer size: %d\n", bufferFrameSize)

	var arc *wca.IAudioRenderClient
	if err := rac.GetService(wca.IID_IAudioRenderClient, &arc); err != nil {
		return err
	}
	defer arc.Release()

	var acc *wca.IAudioCaptureClient
	if err := cac.GetService(wca.IID_IAudioCaptureClient, &acc); err != nil {
		return err
	}
	defer acc.Release()

	if err := rac.Start(); err != nil {
		return err
	}
	defer rac.Stop()
	if err := cac.Start(); err != nil {
		return err
	}
	defer cac.Stop()

	log.Println("Start loopback capturing with shared event driven mode")

	var buf []byte
	var offset int
	var lim int
	var start unsafe.Pointer
	var isCapturing bool = true
	var currentDuration time.Duration
	var data *byte
	var b *byte
	var availableFrameSize uint32
	var flags uint32
	var devicePosition uint64
	var qcpPosition uint64

	errorChan := make(chan error, 1)

	time.Sleep(latency)

	for {
		if !isCapturing {
			close(errorChan)
			break
		}
		go func() {
			errorChan <- watchEvent(ctx, audioReadyEvent)
		}()
		select {
		case <-ctx.Done():
			isCapturing = false
			<-errorChan
		case err := <-errorChan:
			currentDuration = time.Duration(float64(offset) / float64(wfx.WBitsPerSample/8) / float64(wfx.NChannels) / float64(wfx.NSamplesPerSec) * float64(time.Second))
			if err != nil {
				isCapturing = false
				break
			}
			if err := acc.GetBuffer(&data, &availableFrameSize, &flags, &devicePosition, &qcpPosition); err != nil {
				continue
			}
			if availableFrameSize == 0 {
				continue
			}

			start = unsafe.Pointer(data)
			lim = int(availableFrameSize) * int(wfx.NBlockAlign)
			buf = make([]byte, lim)

			for n := 0; n < lim; n++ {
				b = (*byte)(unsafe.Pointer(uintptr(start) + uintptr(n)))
				buf[n] = *b
			}

			offset += lim
			output(Chunk{
				Format:          format,
				CurrentDuration: currentDuration,
				Buffer:          buf,
			})

			if err := acc.ReleaseBuffer(availableFrameSize); err != nil {
				return err
			}
		}
	}
	log.Println("Stop capturing")
	return nil
}

func watchEvent(ctx context.Context, event uintptr) (err error) {
	errorChan := make(chan error, 1)
	go func() {
		errorChan <- eventEmitter(event)
	}()
	select {
	case err = <-errorChan:
		close(errorChan)
		return
	case <-ctx.Done():
		err = ctx.Err()
		return
	}
}

func eventEmitter(event uintptr) (err error) {
	dw := wca.WaitForSingleObject(event, wca.INFINITE)
	if dw != 0 {
		return fmt.Errorf("failed to watch event")
	}
	return
}
