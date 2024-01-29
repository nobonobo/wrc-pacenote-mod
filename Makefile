# 現在の HEAD にタグを付ける
VERSION := $(shell git describe --tags --abbrev=0)
# タグを含むコミットの SHA-1 ハッシュ値を取得
COMMIT_HASH := $(shell git rev-parse --short HEAD)

# バージョン名にコミットハッシュ値を付加
VERSION := $(VERSION)-$(COMMIT_HASH)

NAME=$(basename $(shell go list .))
OUTPUT=$(NAME)-$(VERSION).zip

build:
	cd frontend && npm run build
	go build -tags release .

run:
	go run .

depends:
	curl -LO https://github.com/VOICEVOX/voicevox_core/releases/download/0.15.0-preview.13/download-windows-x64.exe
	download-windows-x64.exe --device cpu --version 0.15.0-preview.13
	cp voicevox_core/onnxruntime.dll ./

pack: build
	mkdir -p dist
	cp -Rf voicevox_core dist/voicevox_core
	cp onnxruntime.dll dist/onnxruntime.dll
	cp wrc-pacenote-mod.exe dist/
	cp base.json dist/base.json
	powershell Compress-Archive -Path dist -Force -DestinationPath $(OUTPUT)
