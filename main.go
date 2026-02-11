package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

type TaskStore struct {
	mu      sync.RWMutex
	tasks   []Task
	nextID  int
	dataDir string
}

func NewTaskStore(dataDir string) (*TaskStore, error) {
	s := &TaskStore{
		tasks:   []Task{},
		nextID:  1,
		dataDir: dataDir,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *TaskStore) dataPath() string {
	return filepath.Join(s.dataDir, "tasks.json")
}

func (s *TaskStore) load() error {
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	f, err := os.Open(s.dataPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open tasks file: %w", err)
	}
	defer f.Close()

	var loaded []Task
	if err := json.NewDecoder(f).Decode(&loaded); err != nil {
		return fmt.Errorf("decode tasks file: %w", err)
	}

	s.tasks = loaded
	maxID := 0
	for _, t := range loaded {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	s.nextID = maxID + 1
	if s.nextID < 1 {
		s.nextID = 1
	}

	return nil
}

func (s *TaskStore) save() error {
	tmpPath := s.dataPath() + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.tasks); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("encode tasks: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.dataPath()); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func (s *TaskStore) List() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Task, len(s.tasks))
	copy(out, s.tasks)
	return out
}

func (s *TaskStore) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, errors.New("title is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task := Task{
		ID:        s.nextID,
		Title:     title,
		Done:      false,
		CreatedAt: time.Now().UTC(),
	}
	s.nextID++
	s.tasks = append(s.tasks, task)

	if err := s.save(); err != nil {
		return Task{}, err
	}

	return task, nil
}

func (s *TaskStore) Toggle(id int) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks[i].Done = !s.tasks[i].Done
			if err := s.save(); err != nil {
				return Task{}, err
			}
			return s.tasks[i], nil
		}
	}

	return Task{}, errors.New("task not found")
}

func (s *TaskStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			return s.save()
		}
	}

	return errors.New("task not found")
}

func main() {
	store, err := NewTaskStore("data")
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"tasks": store.List()})
		case http.MethodPost:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}

			var req struct {
				Title string `json:"title"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}

			t, err := store.Create(req.Title)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			writeJSON(w, http.StatusCreated, t)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/tasks/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}

		id, err := strconv.Atoi(parts[0])
		if err != nil {
			http.Error(w, "invalid task id", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPatch:
			t, err := store.Toggle(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, t)
		case http.MethodDelete:
			if err := store.Delete(id); err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Header().Set("Allow", "PATCH, DELETE")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fileServer := http.FileServer(http.Dir("web"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	addr := ":8080"
	log.Printf("Task app running on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
