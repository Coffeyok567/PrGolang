package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {

	server := "http://localhost:8080"
	scanner := bufio.NewScanner(os.Stdin)

	// Ввод имени игрока
	fmt.Print("Введите имя игрока: ")
	scanner.Scan()
	name := scanner.Text()

	// Регистрация на сервере
	resp, _ := http.Post(server, "text/plain",
		strings.NewReader("register="+name))
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(string(body))

	// Основной цикл игры
	for {
		resp, _ := http.Get(server)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		state := string(body)

		fmt.Println("\n" + state) // Печатаем текущий экран игры

		// Разбор фазы
		if strings.Contains(state, "GAME_OVER") {
			fmt.Println("Игра окончена, вы не можете ходить")
			break
		} else if strings.Contains(state, "ATTACK") {
			// Проверка, можно ли ходить
			if strings.Contains(state, "Ваш ход") || !strings.Contains(state, "ATTACK_DONE") {
				fmt.Print("Выберите атаку (head/body/legs): ")
				scanner.Scan()
				attack := scanner.Text()
				resp, _ := http.Post(server, "text/plain",
					strings.NewReader("attack="+name+":"+attack))
				resp.Body.Close()
			} else {
				fmt.Println("Ожидание хода второго игрока...")
				time.Sleep(2 * time.Second)
			}
		} else if strings.Contains(state, "DEFENSE") {
			if strings.Contains(state, "Ваш ход") || !strings.Contains(state, "DEFENSE_DONE") {
				fmt.Print("Выберите защиту (head/body/legs): ")
				scanner.Scan()
				defense := scanner.Text()
				resp, _ := http.Post(server, "text/plain",
					strings.NewReader("defense="+name+":"+defense))
				resp.Body.Close()
			} else {
				fmt.Println("Ожидание хода второго игрока...")
				time.Sleep(2 * time.Second)
			}
		} else {
			// Любые другие состояния — просто ждём
			time.Sleep(2 * time.Second)
		}
	}
}
