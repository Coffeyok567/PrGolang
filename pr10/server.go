package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type Player struct {
	Name         string
	Attack       string
	Defense      string
	HP           int
	AttackDone   bool
	DefenseDone  bool
}

var (
	players = make(map[string]*Player)
	phase   = "WAIT" // WAIT, ATTACK, DEFENSE, RESULT, GAME_OVER
	result  string
	mutex   sync.Mutex
)

var damageByPart = map[string]int{
	"head": 30,
	"body": 20,
	"legs": 10,
}

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		// ===== POST =====
		if r.Method == http.MethodPost {

			body, _ := io.ReadAll(r.Body)
			msg := string(body)

			// --- Регистрация ---
			if strings.HasPrefix(msg, "register=") {
				name := strings.Split(msg, "=")[1]

				if len(players) >= 2 {
					fmt.Fprint(w, "SERVER_FULL")
					return
				}

				players[name] = &Player{Name: name, HP: 100}

				if len(players) == 2 {
					phase = "ATTACK"
				}

				fmt.Fprint(w, "REGISTERED")
				return
			}

			// --- Если игра окончена ---
			if phase == "GAME_OVER" {
				fmt.Fprint(w, "GAME_OVER")
				return
			}

			// --- Атака ---
			if strings.HasPrefix(msg, "attack=") && phase == "ATTACK" {
				name, value := parseAction(msg)
				p := players[name]

				if p.AttackDone {
					fmt.Fprint(w, "WAIT_YOUR_TURN")
					return
				}

				if !validPart(value) {
					fmt.Fprint(w, "INVALID_INPUT")
					return
				}

				p.Attack = value
				p.AttackDone = true

				if allAttackDone() {
					phase = "DEFENSE"
				}

				fmt.Fprint(w, "OK")
				return
			}

			// --- Защита ---
			if strings.HasPrefix(msg, "defense=") && phase == "DEFENSE" {
				name, value := parseAction(msg)
				p := players[name]

				if p.DefenseDone {
					fmt.Fprint(w, "WAIT_YOUR_TURN")
					return
				}

				if !validPart(value) {
					fmt.Fprint(w, "INVALID_INPUT")
					return
				}

				p.Defense = value
				p.DefenseDone = true

				if allDefenseDone() {
					calcRound()
				}

				fmt.Fprint(w, "OK")
				return
			}
		}

		// ===== GET =====
		fmt.Fprint(w, buildScreen())
	})

	fmt.Println("Сервер запущен :8080")
	http.ListenAndServe(":8080", nil)
}

// ===== ЛОГИКА =====

func parseAction(msg string) (string, string) {
	data := strings.Split(strings.Split(msg, "=")[1], ":")
	return data[0], data[1]
}

func validPart(p string) bool {
	return p == "head" || p == "body" || p == "legs"
}

func allAttackDone() bool {
	if len(players) < 2 {
		return false
	}
	for _, p := range players {
		if !p.AttackDone {
			return false
		}
	}
	return true
}

func allDefenseDone() bool {
	for _, p := range players {
		if !p.DefenseDone {
			return false
		}
	}
	return true
}

func calcRound() {
	var p1, p2 *Player
	for _, p := range players {
		if p1 == nil {
			p1 = p
		} else {
			p2 = p
		}
	}

	result = "=== РЕЗУЛЬТАТ РАУНДА ===\n"

	applyHit(p1, p2)
	applyHit(p2, p1)

	if p1.HP <= 0 || p2.HP <= 0 {
		phase = "GAME_OVER"
		if p1.HP <= 0 {
			result += "\nПОБЕДИЛ " + p2.Name + "\n"
		} else {
			result += "\nПОБЕДИЛ " + p1.Name + "\n"
		}
		return
	}

	clearActions()
	phase = "RESULT"
}

func applyHit(a, d *Player) {
	if a.Attack != d.Defense {
		dmg := damageByPart[a.Attack]
		d.HP -= dmg
		result += fmt.Sprintf("%s ударил %s в %s (-%d)\n",
			a.Name, d.Name, a.Attack, dmg)
	} else {
		result += fmt.Sprintf("%s защитился от %s\n",
			d.Name, a.Name)
	}
}

func clearActions() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
		p.AttackDone = false
		p.DefenseDone = false
	}
}

// ===== ЭКРАН ИГРЫ =====
func buildScreen() string {

	screen := "=== СОСТОЯНИЕ ИГРЫ ===\n"

	for _, p := range players {
		screen += fmt.Sprintf("%s: %d HP\n", p.Name, p.HP)
	}

	screen += "\nФаза: " + phase + "\n\n"

	if result != "" {
		screen += result + "\n"
		result = ""
	}

	if phase == "RESULT" {
		phase = "ATTACK"
	}

	if phase == "GAME_OVER" {
		screen += "\nИГРА ОКОНЧЕНА\n"
	}

	return screen
}
