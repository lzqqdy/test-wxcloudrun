package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/ping", pingHandler)
	mux.HandleFunc("/api/whoami", whoamiHandler)
	mux.HandleFunc("/api/echo", echoHandler)
	mux.HandleFunc("/api/notes", notesHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	addr := ":" + port
	log.Printf("test-wxcloudrun 启动，监听 %s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-WX-SERVICE")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html lang="zh-CN">
<head><meta charset="utf-8"><title>test-wxcloudrun</title></head>
<body>
  <h1>微信云托管测试服务已启动</h1>
  <p>可用接口：</p>
  <ul>
    <li><a href="/health">GET /health</a></li>
    <li><a href="/api/ping">GET /api/ping</a></li>
    <li><a href="/api/whoami">GET /api/whoami</a>（小程序 callContainer 会带上 openid）</li>
    <li>POST /api/echo</li>
    <li>GET/POST /api/notes</li>
  </ul>
</body>
</html>`)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"time": time.Now().Format(time.RFC3339),
	})
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "pong",
		"service": "test-wxcloudrun",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func whoamiHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "从小程序 wx.cloud.callContainer 进来时，下面这些头由微信注入",
		"headers": map[string]string{
			"X-WX-OPENID":      r.Header.Get("X-WX-OPENID"),
			"X-WX-APPID":       r.Header.Get("X-WX-APPID"),
			"X-WX-UNIONID":     r.Header.Get("X-WX-UNIONID"),
			"X-WX-FROM-OPENID": r.Header.Get("X-WX-FROM-OPENID"),
			"X-WX-FROM-APPID":  r.Header.Get("X-WX-FROM-APPID"),
			"X-WX-ENV":         r.Header.Get("X-WX-ENV"),
			"X-WX-SOURCE":      r.Header.Get("X-WX-SOURCE"),
			"X-Forwarded-For":  r.Header.Get("X-Forwarded-For"),
			"User-Agent":       r.Header.Get("User-Agent"),
		},
	})
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST"})
		return
	}
	var body any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"echo":   body,
		"openid": r.Header.Get("X-WX-OPENID"),
	})
}

type note struct {
	ID      int    `json:"id"`
	Text    string `json:"text"`
	OpenID  string `json:"openid"`
	Created string `json:"created"`
}

var (
	notesMu sync.Mutex
	notes   []note
	nextID  = 1
)

func notesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		notesMu.Lock()
		defer notesMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
	case http.MethodPost:
		var req struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "need json {\"text\":\"...\"}"})
			return
		}
		notesMu.Lock()
		item := note{
			ID:      nextID,
			Text:    req.Text,
			OpenID:  r.Header.Get("X-WX-OPENID"),
			Created: time.Now().Format(time.RFC3339),
		}
		nextID++
		notes = append(notes, item)
		notesMu.Unlock()
		writeJSON(w, http.StatusOK, item)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use GET or POST"})
	}
}
