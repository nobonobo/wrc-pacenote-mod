# 現在の HEAD にタグを付ける
VERSION := $(shell git describe --tags --abbrev=0)
# タグを含むコミットの SHA-1 ハッシュ値を取得
COMMIT_HASH := $(shell git rev-parse --short HEAD)

# バージョン名にコミットハッシュ値を付加
LONG_VERSION := $(VERSION)-$(COMMIT_HASH)

NAME=$(notdir $(shell go list .))
OUTPUT=$(NAME)-$(LONG_VERSION).zip

build:
	go generate .
	go build .

run:
	go run -tags develop .

sync: build
	mkdir -p dist/$(VERSION)
	cp wrc-pacenote-mod.exe dist/$(VERSION)/

archive:
	mkdir -p releases
	powershell Compress-Archive -Path dist\\$(VERSION) -Force -DestinationPath releases/$(OUTPUT)

pack: sync archive
