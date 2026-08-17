package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed web/index.html
var indexHTML []byte

func main() {
	go func() {
		if err := initDB(); err != nil {
			log.Printf("启动时数据库未就绪: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/ping", pingHandler)
	mux.HandleFunc("/api/db", dbHandler)
	mux.HandleFunc("/api/whoami", whoamiHandler)
	mux.HandleFunc("/api/echo", echoHandler)
	mux.HandleFunc("GET /api/notes", notesCollectionHandler)
	mux.HandleFunc("POST /api/notes", notesCollectionHandler)
	mux.HandleFunc("GET /api/notes/{id}", noteItemHandler)
	mux.HandleFunc("PUT /api/notes/{id}", noteItemHandler)
	mux.HandleFunc("DELETE /api/notes/{id}", noteItemHandler)

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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
	_, _ = w.Write(indexHTML)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	dbMu.Lock()
	conn := db
	dbMu.Unlock()
	dbOK := conn != nil && conn.Ping() == nil
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"time": time.Now().Format(time.RFC3339),
		"db":   dbOK,
	})
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "pong",
		"service": "test-wxcloudrun",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func dbHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, dbStatus())
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
