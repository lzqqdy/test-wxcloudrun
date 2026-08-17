package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type note struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	OpenID    string `json:"openid"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func notesCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if err := ensureDB(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		listNotes(w)
	case http.MethodPost:
		createNote(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use GET or POST"})
	}
}

func noteItemHandler(w http.ResponseWriter, r *http.Request) {
	if err := ensureDB(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		getNote(w, id)
	case http.MethodPut:
		updateNote(w, r, id)
	case http.MethodDelete:
		deleteNote(w, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use GET, PUT or DELETE"})
	}
}

func listNotes(w http.ResponseWriter) {
	rows, err := db.Query(`SELECT id, text, openid, created_at, updated_at FROM notes ORDER BY id DESC`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := make([]note, 0)
	for rows.Next() {
		item, scanErr := scanNote(rows)
		if scanErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": scanErr.Error()})
			return
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": items})
}

func getNote(w http.ResponseWriter, id int) {
	item, err := loadNote(id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "note not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func createNote(w http.ResponseWriter, r *http.Request) {
	text, ok := readNoteText(w, r)
	if !ok {
		return
	}
	res, err := db.Exec(`INSERT INTO notes (text, openid) VALUES (?, ?)`, text, r.Header.Get("X-WX-OPENID"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	item, err := loadNote(int(id))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func updateNote(w http.ResponseWriter, r *http.Request, id int) {
	text, ok := readNoteText(w, r)
	if !ok {
		return
	}
	res, err := db.Exec(`UPDATE notes SET text = ? WHERE id = ?`, text, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "note not found"})
		return
	}
	item, err := loadNote(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func deleteNote(w http.ResponseWriter, id int) {
	res, err := db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "note not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func loadNote(id int) (note, error) {
	row := db.QueryRow(`SELECT id, text, openid, created_at, updated_at FROM notes WHERE id = ?`, id)
	return scanNote(row)
}

func scanNote(s interface {
	Scan(dest ...any) error
}) (note, error) {
	var item note
	var created, updated time.Time
	err := s.Scan(&item.ID, &item.Text, &item.OpenID, &created, &updated)
	if err != nil {
		return note{}, err
	}
	item.CreatedAt = created.Format(time.RFC3339)
	item.UpdatedAt = updated.Format(time.RFC3339)
	return item, nil
}

func readNoteText(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "need json {\"text\":\"...\"}"})
		return "", false
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return "", false
	}
	if len([]rune(text)) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text too long"})
		return "", false
	}
	return text, true
}
