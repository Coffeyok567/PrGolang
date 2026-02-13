package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Действия игрока за раунд
type PlayerAction struct {
	Attack  string
	Defense string
}

var (
	// Храним действия игроков
	players = make(map[string]PlayerAction)

	// HP игроков
	hp = map[string]int{
		"player1": 100,
		"player2": 100,
	}

	mutex       sync.Mutex
	roundResult string
)

func main() {

	// Один HTTP-обработчик
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// ===== ПОЛУЧЕНИЕ ХОДА ОТ КЛИЕНТА =====
		if r.Method == http.MethodPost {

			// Формат сообщения:
			// player1;attack=head;defense=body
			body, _ := io.ReadAll(r.Body)
			data := strings.Split(string(body), ";")

			player := data[0]
			attack := strings.Split(data[1], "=")[1]
			defense := strings.Split(data[2], "=")[1]

			mutex.Lock()

			// Сохраняем ход игрока
			players[player] = PlayerAction{attack, defense}

			// Если оба игрока сделали ход
			if len(players) == 2 {

				p1 := players["player1"]
				p2 := players["player2"]

				result := "=== РЕЗУЛЬТАТ РАУНДА ===\n"

				// Атака Player1
				if p1.Attack != p2.Defense {
					hp["player2"] -= 20
					result += "Player1 попал по Player2\n"
				} else {
					result += "Player2 защитился от Player1\n"
				}

				// Атака Player2
				if p2.Attack != p1.Defense {
					hp["player1"] -= 20
					result += "Player2 попал по Player1\n"
				} else {
					result += "Player1 защитился от Player2\n"
				}

				// Текущее HP
				result += fmt.Sprintf(
					"\nHP:\nPlayer1 = %d\nPlayer2 = %d\n",
					hp["player1"], hp["player2"],
				)

				// Проверка победителя
				if hp["player1"] <= 0 {
					result += "\nПОБЕДИЛ Player2\n"
				} else if hp["player2"] <= 0 {
					result += "\nПОБЕДИЛ Player1\n"
				}

				roundResult = result

				// Очищаем ходы для нового раунда
				players = make(map[string]PlayerAction)
			}

			mutex.Unlock()
			fmt.Fprint(w, "Ход принят")

		} else {
			// ===== ОТПРАВКА РЕЗУЛЬТАТА КЛИЕНТУ =====
			mutex.Lock()
			fmt.Fprint(w, roundResult)
			mutex.Unlock()
		}
	})

	fmt.Println("PvP сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}
