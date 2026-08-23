package web

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ishaan-Buddharaju/open-gallery/storage"
)

//go:embed gallery.html
var galleryHTML []byte

//go:embed harness.html
var harnessHTML []byte

//go:embed justify.js
var justifyJS []byte

type Server struct {
	db       *sql.DB
	http     *http.Server
	imageDir string
}

func New(db *sql.DB, addr string, imageDir string) *Server {
	s := &Server{db: db, imageDir: imageDir}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleGallery)
	mux.HandleFunc("GET /harness", s.handleHarness)
	mux.HandleFunc("GET /justify.js", s.handleJustifyJS)
	mux.HandleFunc("GET /api/submissions", s.handleSubmissions)
	mux.HandleFunc("GET /images/{path...}", s.handleImage)

	s.http = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	log.Printf("web server listening on %s", s.http.Addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) handleGallery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(galleryHTML)
}

func (s *Server) handleHarness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(harnessHTML)
}

func (s *Server) handleJustifyJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write(justifyJS)
}

type submissionsResponse struct {
	Accepted    int                    `json:"accepted"`
	Submissions []apiSubmission        `json:"submissions"`
}

type apiImage struct {
	URL    string `json:"url"`
	Width  int    `json:"w"`
	Height int    `json:"h"`
}

type apiSubmission struct {
	Author    string     `json:"author"`
	Source    string     `json:"source"`
	Images    []apiImage `json:"images"`
	Caption   string     `json:"caption"`
	Timestamp string     `json:"timestamp"`
}

func (s *Server) handleSubmissions(w http.ResponseWriter, r *http.Request) {
	count, err := storage.CountAccepted(s.db)
	if err != nil {
		log.Printf("CountAccepted: %v", err)
		count = 0
	}

	subs, err := storage.ListAcceptedSubmissions(s.db, 24)
	if err != nil {
		log.Printf("ListAcceptedSubmissions: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	apiSubs := make([]apiSubmission, 0, len(subs))
	for _, sub := range subs {
		var images []apiImage
		if sub.Images != "" {
			var metas []struct {
				Path   string `json:"path"`
				Width  int    `json:"w"`
				Height int    `json:"h"`
			}
			if err := json.Unmarshal([]byte(sub.Images), &metas); err == nil && len(metas) > 0 && metas[0].Path != "" {
				for _, m := range metas {
					images = append(images, apiImage{
						URL:    "/images/" + filepath.Base(m.Path),
						Width:  m.Width,
						Height: m.Height,
					})
				}
			} else {
				var paths []string
				json.Unmarshal([]byte(sub.Images), &paths)
				for _, p := range paths {
					images = append(images, apiImage{URL: "/images/" + filepath.Base(p)})
				}
			}
		}
		apiSubs = append(apiSubs, apiSubmission{
			Author:    sub.Author,
			Source:    sub.Source,
			Images:    images,
			Caption:   sub.Caption,
			Timestamp: sub.Timestamp,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submissionsResponse{
		Accepted:    count,
		Submissions: apiSubs,
	})
}

func (s *Server) handleImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("path")
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.imageDir, name))
}
