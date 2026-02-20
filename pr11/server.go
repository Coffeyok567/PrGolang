package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	players     = make(map[string]bool)
	gameState   = "WAIT"
	currentTurn = ""

	chatHistory []string
	mutex       sync.Mutex
)

func main() {
	http.HandleFunc("/pvp", pvpHandler)
	http.HandleFunc("/chat", chatHandler)

	fmt.Println("PVP + CHAT сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}

func pvpHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		msg := string(body)

		// ===== регистрация =====
		if strings.HasPrefix(msg, "register=") {
			name := strings.TrimPrefix(msg, "register=")
			players[name] = true

			if len(players) == 1 {
				currentTurn = name
				gameState = "ATTACK"
			}

			fmt.Fprint(w, "OK")
			return
		}

		// ===== атака =====
		if strings.HasPrefix(msg, "attack=") {
			gameState = "DEFENSE"
			fmt.Fprint(w, "OK")
			return
		}

		// ===== защита =====
		if strings.HasPrefix(msg, "defense=") {
			gameState = "ATTACK"
			fmt.Fprint(w, "OK")
			return
		}
	}

	// ===== GET =====
	if len(players) < 2 {
		fmt.Fprint(w, "WAIT")
		return
	}

	fmt.Fprint(w, gameState)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		chatHistory = append(chatHistory, string(body))

		if len(chatHistory) > 50 {
			chatHistory = chatHistory[len(chatHistory)-50:]
		}
		return
	}

	for _, msg := range chatHistory {
		fmt.Fprintln(w, msg)
	}
}
