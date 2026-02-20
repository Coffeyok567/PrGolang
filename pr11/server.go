package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	players       = make(map[string]bool)
	currentTurn   = ""
	gameState     = "WAIT"
	attacks       = make(map[string]string)
	defenses      = make(map[string]string)

	chatHistory []string
	mutex       sync.Mutex
)

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("PVP + CHAT сервер запущен :8080")
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		msg := string(body)

		// ===== РЕГИСТРАЦИЯ =====
		if strings.HasPrefix(msg, "register=") {
			name := strings.TrimPrefix(msg, "register=")
			players[name] = true
			if currentTurn == "" {
				currentTurn = name
			}
			fmt.Fprint(w, "OK")
			return
		}

		// ===== АТАКА =====
		if strings.HasPrefix(msg, "attack=") {
			data := strings.TrimPrefix(msg, "attack=")
			parts := strings.Split(data, ":")
			attacks[parts[0]] = parts[1]
			gameState = "DEFENSE"
			return
		}

		// ===== ЗАЩИТА =====
		if strings.HasPrefix(msg, "defense=") {
			data := strings.TrimPrefix(msg, "defense=")
			parts := strings.Split(data, ":")
			defenses[parts[0]] = parts[1]
			gameState = "ATTACK"
			return
		}

		// ===== ЧАТ =====
		if strings.HasPrefix(msg, "chat=") {
			text := strings.TrimPrefix(msg, "chat=")
			chatHistory = append(chatHistory, text)
			if len(chatHistory) > 30 {
				chatHistory = chatHistory[len(chatHistory)-30:]
			}
			return
		}
	}

	// ===== GET =====
	if r.Method == http.MethodGet {
		if len(players) < 2 {
			fmt.Fprint(w, "WAIT")
			return
		}
		fmt.Fprint(w, gameState)
	}
}
