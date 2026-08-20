package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:18791", "local UI address")
	noOpen := flag.Bool("no-open", false, "do not open a browser")
	flag.Parse()

	path, err := defaultStorePath()
	if err != nil {
		log.Fatal(err)
	}
	sess := newSession(&fileStore{path: path})
	sess.start()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(uiHTML)
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, sess.snapshot())
	})
	mux.HandleFunc("/api/pair", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var req struct{ Host, Code string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, 400, "invalid json")
			return
		}
		req.Code = strings.TrimSpace(req.Code)
		if len(req.Code) != 6 {
			writeErr(w, 400, "enter the 6-digit code from the robot")
			return
		}
		if err := sess.pair(req.Host, req.Code); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/unpair", func(w http.ResponseWriter, r *http.Request) {
		sess.unpair()
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/pause", func(w http.ResponseWriter, r *http.Request) {
		sess.mu.Lock()
		sess.paused = !sess.paused
		paused := sess.paused
		sess.mu.Unlock()
		writeJSON(w, map[string]any{"paused": paused})
	})

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + ln.Addr().String()
	fmt.Fprintf(os.Stderr, "Kestrel Buddy (%s)\nOpen %s  then pair with the code from the robot.\n", currentOS(), url)
	if !*noOpen {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openLocal(url)
		}()
	}
	log.Fatal(http.Serve(ln, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func openLocal(u string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	_ = cmd.Start()
}
