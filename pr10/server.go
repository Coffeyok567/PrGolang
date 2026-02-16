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
	phase   = "WAIT" // WAIT, ATTACK, DEFENSE, RESULT, GAME_OVER
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

			// --- Если игра окончена — запрет любых действий ---
			if phase == "GAME_OVER" {
				fmt.Fprint(w, "GAME_OVER")
				return
			}

			// --- Атака ---
			if strings.HasPrefix(msg, "attack=") && phase == "ATTACK" {
				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				players[parts[0]].Attack = parts[1]

				if allAttacks() {
					phase = "DEFENSE"
				}

				fmt.Fprint(w, "OK")
				return
			}

			// --- Защита ---
			if strings.HasPrefix(msg, "defense=") && phase == "DEFENSE" {
				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				players[parts[0]].Defense = parts[1]

				if allDefenses() {
					calcResult()
				}

				fmt.Fprint(w, "OK")
				return
			}
		}

		// ===== GET =====
		fmt.Fprint(w, buildStatus())
	})

	fmt.Println("Сервер запущен :8080")
	http.ListenAndServe(":8080", nil)
}

// ===== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ =====

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

// ===== РАСЧЁТ РАУНДА =====
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

	applyHit(p1, p2)
	applyHit(p2, p1)

	// Проверка окончания игры
	if p1.HP <= 0 || p2.HP <= 0 {
		phase = "GAME_OVER"
		if p1.HP <= 0 {
			result += "\nПОБЕДИЛ " + p2.Name + "\n"
		} else {
			result += "\nПОБЕДИЛ " + p1.Name + "\n"
		}
		return
	}

	phase = "RESULT"
	clearActions()
}

// Нанесение урона
func applyHit(attacker, defender *Player) {
	if attacker.Attack != defender.Defense {
		dmg := damageByPart[attacker.Attack]
		defender.HP -= dmg
		result += fmt.Sprintf(
			"%s ударил %s в %s (-%d HP)\n",
			attacker.Name, defender.Name, attacker.Attack, dmg,
		)
	} else {
		result += fmt.Sprintf(
			"%s защитился от удара %s\n",
			defender.Name, attacker.Name,
		)
	}
}

// Очистка атак и защит
func clearActions() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
}

// ===== ФОРМИРОВАНИЕ СТАТУСА =====
func buildStatus() string {

	status := "=== СОСТОЯНИЕ ИГРЫ ===\n"

	for _, p := range players {
		status += fmt.Sprintf("%s: %d HP\n", p.Name, p.HP)
	}

	status += "\nФаза: " + phase + "\n\n"

	if phase == "RESULT" {
		phase = "ATTACK"
		return result + "\n" + status
	}

	if phase == "GAME_OVER" {
		return result + "\n" + status + "\nИгра окончена\n"
	}

	return status
}
