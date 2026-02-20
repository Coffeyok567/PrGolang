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

			// --- ЛОГИКА ЧАТА ---
			if strings.HasPrefix(msg, "[CHAT]") {
				chat_history = append(chat_history, strings.TrimPrefix(msg, "[CHAT]"))
				fmt.Println("Новое сообщение в чате:", msg)
				fmt.Fprint(w, "OK")
				return
			}

			// --- ЛОГИКА PVP ---
			if strings.HasPrefix(msg, "register=") {
				name := strings.Split(msg, "=")[1]
				if len(players) < 2 {
					players[name] = &Player{Name: name, HP: 100}
					if len(players) == 2 {
						phase = "ATTACK"
					}
					fmt.Fprint(w, "REGISTERED")
				} else {
					fmt.Fprint(w, "SERVER_FULL")
				}
				return
			}

			if strings.HasPrefix(msg, "attack=") && phase == "ATTACK" {
				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				if p, ok := players[parts[0]]; ok {
					p.Attack = parts[1]
				}
				if allReady("attack") {
					phase = "DEFENSE"
				}
				fmt.Fprint(w, "OK")
				return
			}

			if strings.HasPrefix(msg, "defense=") && phase == "DEFENSE" {
				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				if p, ok := players[parts[0]]; ok {
					p.Defense = parts[1]
				}
				if allReady("defense") {
					calcResult()
					phase = "RESULT"
				}
				fmt.Fprint(w, "OK")
				return
			}
		} else {
			// GET ЗАПРОС: Возвращаем состояние игры + чат через разделитель
			gameStatus := phase
			if phase == "RESULT" {
				gameStatus = result
				resetRound()
			}

			chatData := strings.Join(chat_history, "\n")
			fmt.Fprintf(w, "%s|---CHAT---|%s", gameStatus, chatData)
		}
	})

	fmt.Println("Сервер запущен на :8080. Ожидание игроков...")
	http.ListenAndServe(":8080", nil)
}

func allReady(mode string) bool {
	if len(players) < 2 {
		return false
	}
	for _, p := range players {
		if mode == "attack" && p.Attack == "" {
			return false
		}
		if mode == "defense" && p.Defense == "" {
			return false
		}
	}
	return true
}

func calcResult() {
	var ps []*Player
	for _, p := range players {
		ps = append(ps, p)
	}
	p1, p2 := ps[0], ps[1]

	result = "=== РЕЗУЛЬТАТ РАУНДА ===\n"
	applyDamage(p1, p2)
	applyDamage(p2, p1)
	result += fmt.Sprintf("\nHP: %s(%d) | %s(%d)\n", p1.Name, p1.HP, p2.Name, p2.HP)
}

func applyDamage(att, def *Player) {
	if att.Attack != def.Defense {
		dmg := damageByPart[att.Attack]
		def.HP -= dmg
		result += fmt.Sprintf("%s попал в %s (-%d HP)\n", att.Name, att.Attack, dmg)
	} else {
		result += fmt.Sprintf("%s заблокировал удар в %s\n", def.Name, att.Attack)
	}
}

func resetRound() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
	phase = "ATTACK"
}
