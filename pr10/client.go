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

	// Выбор игрока
	fmt.Print("Введите игрока (player1 / player2): ")
	scanner.Scan()
	player := scanner.Text()

	for {

		// Выбор атаки
		fmt.Print("Атаковать (head/body/legs): ")
		scanner.Scan()
		attack := scanner.Text()

		// Выбор защиты
		fmt.Print("Защищать (head/body/legs): ")
		scanner.Scan()
		defense := scanner.Text()

		// Отправка хода на сервер
		data := player + ";attack=" + attack + ";defense=" + defense
		http.Post(server, "text/plain", strings.NewReader(data))

		// Небольшая пауза, ждём второго игрока
		time.Sleep(2 * time.Second)

		// Получаем результат раунда
		resp, _ := http.Get(server)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Если есть результат — выводим
		if string(body) != "" {
			fmt.Println("\n" + string(body))
		}
	}
}
