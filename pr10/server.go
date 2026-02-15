package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Информация об игроке
type Player struct {
	Name    string
	Attack  string
	Defense string
	HP      int
}

var (
	players = make(map[string]*Player) // name -> player
	phase   = "attack"                  // attack или defense
	result  string

	mutex sync.Mutex
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		mutex.Lock()
		defer mutex.Unlock()

		// ===== РЕГИСТРАЦИЯ ИГРОКА =====
		if r.Method == http.MethodPost {

			body, _ := io.ReadAll(r.Body)
			data := string(body)

			// register=Имя
			if strings.HasPrefix(data, "register=") {

				name := strings.Split(data, "=")[1]

				if len(players) >= 2 {
					fmt.Fprint(w, "Сервер заполнен (2 игрока)")
					return
				}

				players[name] = &Player{
					Name: name,
					HP:   100,
				}

				fmt.Fprint(w, "Вы зарегистрированы как "+name)
				return
			}

			// attack=Имя:head
			if strings.HasPrefix(data, "attack=") && phase == "attack" {

				parts := strings.Split(strings.Split(data, "=")[1], ":")
				name := parts[0]
				attack := parts[1]

				players[name].Attack = attack

				if allAttacksDone() {
					phase = "defense"
					result = "Все выбрали атаку. Теперь защита.\n"
				}

				fmt.Fprint(w, "Атака принята")
				return
			}

			// defense=Имя:body
			if strings.HasPrefix(data, "defense=") && phase == "defense" {

				parts := strings.Split(strings.Split(data, "=")[1], ":")
				name := parts[0]
				defense := parts[1]

				players[name].Defense = defense

				if allDefensesDone() {
					calcRound()
					phase = "attack"
					clearActions()
				}

				fmt.Fprint(w, "Защита принята")
				return
			}

		} else {
			// ===== ОТПРАВКА СОСТОЯНИЯ =====
			fmt.Fprint(w, result)
		}
	})

	fmt.Println("HTTP PvP сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}

// Проверка — все ли выбрали атаку
func allAttacksDone() bool {
	for _, p := range players {
		if p.Attack == "" {
			return false
		}
	}
	return len(players) == 2
}

// Проверка — все ли выбрали защиту
func allDefensesDone() bool {
	for _, p := range players {
		if p.Defense == "" {
			return false
		}
	}
	return true
}

// Расчёт раунда
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

	if p1.Attack != p2.Defense {
		p2.HP -= 20
		result += p1.Name + " попал по " + p2.Name + "\n"
	} else {
		result += p2.Name + " защитился от " + p1.Name + "\n"
	}

	if p2.Attack != p1.Defense {
		p1.HP -= 20
		result += p2.Name + " попал по " + p1.Name + "\n"
	} else {
		result += p1.Name + " защитился от " + p2.Name + "\n"
	}

	result += fmt.Sprintf(
		"\nHP:\n%s = %d\n%s = %d\n",
		p1.Name, p1.HP,
		p2.Name, p2.HP,
	)

	if p1.HP <= 0 {
		result += "\nПОБЕДИЛ " + p2.Name + "\n"
	}
	if p2.HP <= 0 {
		result += "\nПОБЕДИЛ " + p1.Name + "\n"
	}
}

// Очистка действий для нового раунда
func clearActions() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
}
