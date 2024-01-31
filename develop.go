//go:build develop

package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sync"
)

func setup(ctx context.Context) {
	u, _ := url.Parse("http://127.0.0.1:8088")
	host, port, _ := net.SplitHostPort(u.Host)
	cmd := (*exec.Cmd)(nil)
	boot := sync.OnceFunc(func() {
		cmd = exec.CommandContext(ctx, "npm", "run", "dev", "--",
			"--host="+host, "--port="+port)
		cmd.Dir = "frontend"
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Println("dev server:", "started")
		go func() {
			<-ctx.Done()
			log.Println("dev server:", "stopping")
			cmd.Cancel()
			cmd.Wait()
			cmd = nil
		}()
		if err := cmd.Start(); err != nil {
			log.Print(err)
		}
		for {
			res, err := http.Get(u.String())
			if err != nil {
				continue
			}
			res.Body.Close()
			break
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		boot()
		httputil.NewSingleHostReverseProxy(u).ServeHTTP(w, r)
	})
}
