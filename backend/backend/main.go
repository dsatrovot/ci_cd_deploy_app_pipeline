package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Service struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	Status        string     `json:"status"`
	StatusCode    int        `json:"status_code"`
	ResponseMS    *int64     `json:"response_ms"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type App struct {
	db         *sql.DB
	httpClient *http.Client
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://monitor:monitor@localhost:5432/servicemonitor?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil { log.Fatal(err) }
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil { log.Fatalf("database connection failed: %v", err) }
	if err := migrate(ctx, db); err != nil { log.Fatalf("database migration failed: %v", err) }

	app := &App{db: db, httpClient: &http.Client{Timeout: 5 * time.Second}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", app.health)
	mux.HandleFunc("/api/services", app.services)
	mux.HandleFunc("/api/services/", app.serviceAction)

	addr := ":8080"
	log.Printf("service-monitor backend listening on %s", addr)
	server := &http.Server{Addr: addr, Handler: cors(mux), ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}

func migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS services (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL CHECK (length(trim(name)) > 0),
  url TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'unknown' CHECK (status IN ('unknown','up','down')),
  status_code INTEGER NOT NULL DEFAULT 0,
  response_ms BIGINT,
  last_checked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`)
	return err
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { methodNotAllowed(w, "GET"); return }
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status":"error", "database":"unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status":"ok", "database":"ok"})
}

func (a *App) services(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.QueryContext(r.Context(), `SELECT id,name,url,status,status_code,response_ms,last_checked_at,created_at FROM services ORDER BY id DESC`)
		if err != nil { writeError(w, 500, "database query failed"); return }
		defer rows.Close()
		result := []Service{}
		for rows.Next() {
			var s Service
			var response sql.NullInt64
			var checked sql.NullTime
			if err := rows.Scan(&s.ID,&s.Name,&s.URL,&s.Status,&s.StatusCode,&response,&checked,&s.CreatedAt); err != nil {
				writeError(w, 500, "database read failed"); return
			}
			if response.Valid { v := response.Int64; s.ResponseMS = &v }
			if checked.Valid { v := checked.Time; s.LastCheckedAt = &v }
			result = append(result, s)
		}
		if err := rows.Err(); err != nil { writeError(w, 500, "database read failed"); return }
		writeJSON(w, 200, result)
	case http.MethodPost:
		var input struct { Name string `json:"name"`; URL string `json:"url"` }
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil { writeError(w, 400, "invalid JSON"); return }
		input.Name = strings.TrimSpace(input.Name)
		parsed, err := url.ParseRequestURI(strings.TrimSpace(input.URL))
		if input.Name == "" || err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			writeError(w, 400, "name and a valid http(s) URL are required"); return
		}
		var s Service
		err = a.db.QueryRowContext(r.Context(), `INSERT INTO services(name,url) VALUES($1,$2) RETURNING id,name,url,status,status_code,response_ms,last_checked_at,created_at`, input.Name, parsed.String()).
			Scan(&s.ID,&s.Name,&s.URL,&s.Status,&s.StatusCode,&s.ResponseMS,&s.LastCheckedAt,&s.CreatedAt)
		if err != nil { writeError(w, 500, "could not save service"); return }
		writeJSON(w, 201, s)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func (a *App) serviceAction(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/services/"), "/")
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 { writeError(w, 400, "invalid service id"); return }
	if len(parts) == 2 && parts[1] == "check" {
		if r.Method != http.MethodPost { methodNotAllowed(w, "POST"); return }
		a.checkService(w, r, id)
		return
	}
	if len(parts) != 1 { http.NotFound(w,r); return }
	if r.Method != http.MethodDelete { methodNotAllowed(w, "DELETE"); return }
	res, err := a.db.ExecContext(r.Context(), `DELETE FROM services WHERE id=$1`, id)
	if err != nil { writeError(w, 500, "could not delete service"); return }
	n, _ := res.RowsAffected()
	if n == 0 { writeError(w, 404, "service not found"); return }
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) checkService(w http.ResponseWriter, r *http.Request, id int64) {
	var name, target string
	err := a.db.QueryRowContext(r.Context(), `SELECT name,url FROM services WHERE id=$1`, id).Scan(&name,&target)
	if errors.Is(err, sql.ErrNoRows) { writeError(w,404,"service not found"); return }
	if err != nil { writeError(w,500,"could not load service"); return }

	// Avoid turning the demo into an open proxy: only public HTTP(S) URLs are accepted at registration.
	start := time.Now()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	statusCode, status := 0, "down"
	var elapsed int64
	if err == nil {
		resp, reqErr := a.httpClient.Do(req)
		elapsed = time.Since(start).Milliseconds()
		if reqErr == nil {
			statusCode = resp.StatusCode
			resp.Body.Close()
			if statusCode >= 200 && statusCode < 400 { status = "up" }
		}
	}
	_, err = a.db.ExecContext(r.Context(), `UPDATE services SET status=$1,status_code=$2,response_ms=$3,last_checked_at=NOW() WHERE id=$4`, status,statusCode,elapsed,id)
	if err != nil { writeError(w,500,"could not save check result"); return }
	var s Service
	var response sql.NullInt64
	var checked sql.NullTime
	err = a.db.QueryRowContext(r.Context(), `SELECT id,name,url,status,status_code,response_ms,last_checked_at,created_at FROM services WHERE id=$1`, id).
		Scan(&s.ID,&s.Name,&s.URL,&s.Status,&s.StatusCode,&response,&checked,&s.CreatedAt)
	if err != nil { writeError(w,500,"could not read check result"); return }
	if response.Valid { v := response.Int64; s.ResponseMS = &v }
	if checked.Valid { v := checked.Time; s.LastCheckedAt = &v }
	writeJSON(w,200,s)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		if os.Getenv("CORS_ORIGIN") == "" { w.Header().Set("Access-Control-Allow-Origin", "*") }
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
		next.ServeHTTP(w,r)
	})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type","application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, message string) { writeJSON(w,status,map[string]string{"error":message}) }
func methodNotAllowed(w http.ResponseWriter, allow string) { w.Header().Set("Allow",allow); writeError(w,405,"method not allowed") }

var _ = fmt.Sprintf
