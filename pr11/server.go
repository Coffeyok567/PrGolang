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
	phase        = "WAIT" // WAIT, ATTACK, DEFENSE, RESULT
	result       string
	mutex        sync.Mutex
)

var damageByPart = map[string]int{"head": 30, "body": 20, "legs": 10}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			msg := string(body)

			// Логика чата
			if strings.HasPrefix(msg, "[CHAT]") {
				chat_history = append(chat_history, strings.TrimPrefix(msg, "[CHAT]"))
				if len(chat_history) > 10 { // Храним последние 10 строк
					chat_history = chat_history[1:]
				}
				return
			}

			// Регистрация в PvP
			if strings.HasPrefix(msg, "register=") {
				name := strings.Split(msg, "=")[1]
				if len(players) < 2 {
					players[name] = &Player{Name: name, HP: 100}
					if len(players) == 2 { phase = "ATTACK" }
				}
				return
			}

			// Ход атаки
			if strings.HasPrefix(msg, "attack=") && phase == "ATTACK" {
				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				if p, ok := players[parts[0]]; ok { p.Attack = parts[1] }
				if allReady("attack") { phase = "DEFENSE" }
				return
			}

			// Ход защиты
			if strings.HasPrefix(msg, "defense=") && phase == "DEFENSE" {
				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				if p, ok := players[parts[0]]; ok { p.Defense = parts[1] }
				if allReady("defense") {
					calcResult()
					phase = "RESULT"
				}
				return
			}
		} else {
			// GET: Отдаем состояние и чат
			status := phase
			if phase == "RESULT" {
				status = result
				// Сброс через ответ (упрощенно)
				go func() {
					fmt.Scanln() // Ждем любого ввода на сервере или просто таймер
					mutex.Lock()
					resetRound()
					mutex.Unlock()
				}()
			}
			chatData := strings.Join(chat_history, "\n")
			fmt.Fprintf(w, "%s|||%s", status, chatData)
		}
	})

	fmt.Println("Сервер запущен на :8080...")
	http.ListenAndServe(":8080", nil)
}

func allReady(mode string) bool {
	if len(players) < 2 { return false }
	for _, p := range players {
		if mode == "attack" && p.Attack == "" { return false }
		if mode == "defense" && p.Defense == "" { return false }
	}
	return true
}

func calcResult() {
	var ps []*Player
	for _, p := range players { ps = append(ps, p) }
	p1, p2 := ps[0], ps[1]
	result = "=== РЕЗУЛЬТАТ РАУНДА ===\n"
	applyDamage(p1, p2)
	applyDamage(p2, p1)
	result += fmt.Sprintf("\nHP: %s(%d) | %s(%d)\n(Нажмите Enter на сервере для след. раунда)", p1.Name, p1.HP, p2.Name, p2.HP)
}

func applyDamage(att, def *Player) {
	if att.Attack != def.Defense {
		dmg := damageByPart[att.Attack]
		def.HP -= dmg
		result += fmt.Sprintf("%s ударил в %s (-%d HP)\n", att.Name, att.Attack, dmg)
	} else {
		result += fmt.Sprintf("%s заблокировал удар в %s\n", def.Name, att.Attack)
	}
}

func resetRound() {
	for _, p := range players {
		p.Attack = ""; p.Defense = ""
	}
	phase = "ATTACK"
}
