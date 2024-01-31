//go:build !develop

package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:generate sh -c "cd frontend && npm run build"

//go:embed all:frontend/build/*
var assets embed.FS

func setup(ctx context.Context) {
	sub, err := fs.Sub(assets, "frontend/build")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", http.FileServer(http.FS(sub)))
}
