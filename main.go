package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
)

type Todo struct {
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	IsComplete  bool      `json:"isComplete"`
	Id          uint64    `json:"id"`
}

type router struct{}

var todos []Todo
var currentId uint64
var mu sync.Mutex

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func error404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func listTodos(w http.ResponseWriter, r *http.Request) {
	res, err := json.Marshal(todos)
	if err != nil {
		log.Fatalf("Error in listTodos: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(res)
	return
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	err := json.NewDecoder(r.Body).Decode(&todo)
	if err != nil {
		log.Fatalf("Error in createTodo: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
	}
	fmt.Printf("todo1 %+v\n", todo)
	todo.CreatedAt = time.Now()
	todo.UpdatedAt = time.Now()
	// todo.expiresAt, err = time.Parse(time.RFC3339, todo.expiresAt)
	if err != nil {
		log.Fatalf("Error parsing expiration time: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mu.Lock()
	currentId = currentId + 1
	mu.Unlock()

	todo.Id = currentId
	todo.IsComplete = false
	fmt.Printf("%+v\n", todo)
	todos = append(todos, todo)
	res, err := json.Marshal(todo)
	if err != nil {
		log.Fatalf("Error in createTodo: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(res)
	return
}

func getTodoById(w http.ResponseWriter, r *http.Request) {
	pathSplit := strings.Split(r.URL.Path, "/")
	id, err := strconv.Atoi(pathSplit[len(pathSplit)-1])
	if err != nil {
		log.Fatalf("Error in getTodoById: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, todo := range todos {
		if todo.Id == uint64(id) {
			res, err := json.Marshal(todo)
			if err != nil {
				log.Fatalf("Error in getTodoById: %s\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(res)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	return
}

func (ro router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	todosPath := regexp.MustCompile(`^\/api\/v1\/todos[\/]*$`)
	todosIdPath := regexp.MustCompile(`^\/api\/v1\/todos\/[0-9]+[\/]*$`)

	switch {
	case path == "/healthz":
		if r.Method == "GET" {
			healthCheck(w, r)
		}
	case todosPath.MatchString(path):
		switch r.Method {
		case "GET":
			listTodos(w, r)
		case "POST":
			createTodo(w, r)
		}
	case todosIdPath.MatchString(path):
		switch r.Method {
		case "GET":
			getTodoById(w, r)
		}
	default:
		error404(w, r)
	}
}

func server() {
	var ro router
	port, hasPort := os.LookupEnv("PORT")
	if !hasPort {
		port = "8000"
	}
	fmt.Printf("Starting server on port %s\n", port)
	sPort := fmt.Sprintf(":%s", port)

	wrappedH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		m := httpsnoop.CaptureMetrics(ro, w, r)
		log.SetFlags(0)
		log.Printf(
			"[%s] - \"%s %s\" %d %dms (%d bytes)",
			t.Format(time.RFC3339),
			r.Method,
			r.URL,
			m.Code,
			m.Duration.Milliseconds(),
			m.Written,
		)
	})

	currentId = 0
	log.Fatal(http.ListenAndServe(sPort, wrappedH))
}

func main() {
	server()
}
