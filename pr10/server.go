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
	players = make(map[string]*Player)
	phase   = "WAIT" // WAIT, ATTACK, DEFENSE, RESULT
	result  string
	mutex   sync.Mutex
)

// Урон по частям тела
var damageByPart = map[string]int{
	"head": 30,
	"body": 20,
	"legs": 10,
}

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		mutex.Lock()
		defer mutex.Unlock()

		if r.Method == http.MethodPost {

			body, _ := io.ReadAll(r.Body)
			msg := string(body)

			// ===== РЕГИСТРАЦИЯ =====
			if strings.HasPrefix(msg, "register=") {
				name := strings.Split(msg, "=")[1]

				if len(players) >= 2 {
					fmt.Fprint(w, "SERVER_FULL")
					return
				}

				players[name] = &Player{
					Name: name,
					HP:   100,
				}

				if len(players) == 2 {
					phase = "ATTACK"
				}

				fmt.Fprint(w, "REGISTERED")
				return
			}

			// ===== АТАКА =====
			if strings.HasPrefix(msg, "attack=") {
				if phase != "ATTACK" {
					fmt.Fprint(w, "WAIT")
					return
				}

				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				players[parts[0]].Attack = parts[1]

				if allAttacks() {
					phase = "DEFENSE"
				}

				fmt.Fprint(w, "OK")
				return
			}

			// ===== ЗАЩИТА =====
			if strings.HasPrefix(msg, "defense=") {
				if phase != "DEFENSE" {
					fmt.Fprint(w, "WAIT")
					return
				}

				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				players[parts[0]].Defense = parts[1]

				if allDefenses() {
					calcResult()
					phase = "RESULT"
				}

				fmt.Fprint(w, "OK")
				return
			}
		}

		// ===== GET: СОСТОЯНИЕ =====
		if phase == "WAIT" {
			fmt.Fprint(w, "WAIT")
		} else if phase == "ATTACK" {
			fmt.Fprint(w, "ATTACK")
		} else if phase == "DEFENSE" {
			fmt.Fprint(w, "DEFENSE")
		} else if phase == "RESULT" {
			fmt.Fprint(w, result)
			resetRound()
		}
	})

	fmt.Println("Сервер запущен :8080")
	http.ListenAndServe(":8080", nil)
}

func allAttacks() bool {
	if len(players) < 2 {
		return false
	}
	for _, p := range players {
		if p.Attack == "" {
			return false
		}
	}
	return true
}

func allDefenses() bool {
	for _, p := range players {
		if p.Defense == "" {
			return false
		}
	}
	return true
}

// ===== РАСЧЁТ УРОНА =====
func calcResult() {

	var p1, p2 *Player
	for _, p := range players {
		if p1 == nil {
			p1 = p
		} else {
			p2 = p
		}
	}

	result = "=== РЕЗУЛЬТАТ РАУНДА ===\n"

	// Атака p1
	if p1.Attack != p2.Defense {
		dmg := damageByPart[p1.Attack]
		p2.HP -= dmg
		result += fmt.Sprintf(
			"%s ударил %s в %s (-%d HP)\n",
			p1.Name, p2.Name, p1.Attack, dmg,
		)
	} else {
		result += fmt.Sprintf(
			"%s защитился от удара %s\n",
			p2.Name, p1.Name,
		)
	}

	// Атака p2
	if p2.Attack != p1.Defense {
		dmg := damageByPart[p2.Attack]
		p1.HP -= dmg
		result += fmt.Sprintf(
			"%s ударил %s в %s (-%d HP)\n",
			p2.Name, p1.Name, p2.Attack, dmg,
		)
	} else {
		result += fmt.Sprintf(
			"%s защитился от удара %s\n",
			p1.Name, p2.Name,
		)
	}

	result += fmt.Sprintf(
		"\nHP:\n%s = %d\n%s = %d\n",
		p1.Name, p1.HP,
		p2.Name, p2.HP,
	)
}

func resetRound() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
	phase = "ATTACK"
}
