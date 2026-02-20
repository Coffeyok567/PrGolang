package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type Player struct {
	Name    string
	Attack  string
	Defense string
	HP      int
}

var (
	players      = make(map[string]*Player)
	chat_history []string
	phase        = "WAIT"
	result       string
	mutex        sync.Mutex
)

var damageByPart = map[string]int{"head": 30, "body": 20, "legs": 10}

func main() {
	http.HandleFunc("/chat", handleChat)
	http.HandleFunc("/game", handleGame)

	fmt.Println("=== СЕРВЕР ЗАПУЩЕН :8080 ===")
	http.ListenAndServe(":8080", nil)
}

// Обработка только чата
func handleChat(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		chat_history = append(chat_history, string(body))
		fmt.Fprint(w, "OK")
	} else {
		fmt.Fprint(w, strings.Join(chat_history, "\n"))
	}
}

// Обработка только PVP
func handleGame(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		msg := string(body)

		if strings.HasPrefix(msg, "register=") {
			name := strings.Split(msg, "=")[1]
			if len(players) < 2 {
				players[name] = &Player{Name: name, HP: 100}
				if len(players) == 2 { phase = "ATTACK" }
			}
		} else if strings.HasPrefix(msg, "attack=") {
			parts := strings.Split(strings.Split(msg, "=")[1], ":")
			if p, ok := players[parts[0]]; ok { p.Attack = parts[1] }
			if allReady("attack") { phase = "DEFENSE" }
		} else if strings.HasPrefix(msg, "defense=") {
			parts := strings.Split(strings.Split(msg, "=")[1], ":")
			if p, ok := players[parts[0]]; ok { p.Defense = parts[1] }
			if allReady("defense") { calcResult(); phase = "RESULT" }
		}
	} else {
		if phase == "RESULT" {
			fmt.Fprint(w, result)
			resetRound()
		} else {
			fmt.Fprint(w, phase)
		}
	}
}

func allReady(m string) bool {
	if len(players) < 2 { return false }
	for _, p := range players {
		if (m == "attack" && p.Attack == "") || (m == "defense" && p.Defense == "") { return false }
	}
	return true
}

func calcResult() {
	var ps []*Player
	for _, p := range players { ps = append(ps, p) }
	p1, p2 := ps[0], ps[1]
	result = fmt.Sprintf("--- РЕЗУЛЬТАТ ---\n%s: A(%s) D(%s)\n%s: A(%s) D(%s)\n", p1.Name, p1.Attack, p1.Defense, p2.Name, p2.Attack, p2.Defense)
	applyDmg(p1, p2); applyDmg(p2, p1)
	result += fmt.Sprintf("HP: %s=%d, %s=%d", p1.Name, p1.HP, p2.Name, p2.HP)
}

func applyDmg(a, d *Player) { if a.Attack != d.Defense { d.HP -= damageByPart[a.Attack] } }

func resetRound() {
	for _, p := range players { p.Attack = ""; p.Defense = "" }
	phase = "ATTACK"
}
