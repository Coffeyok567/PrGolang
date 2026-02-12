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

// Канал для вывода сообщений в консоль
var display_chan = make(chan string, 5)

func main() {

	// Адрес сервера
	server := "http://localhost:8080" // вот тут большой шаурма (ссылку на развернутый сервер на гит хабе - spacecode )

	// Горутина для печати сообщений из канала
	go func() {
		for text_to_print := range display_chan {
			fmt.Println(text_to_print)
		}
	}()

	// Горутина для периодического запроса истории чата с сервера
	go func() {
		last_count := 0 // сколько сообщений уже показали

		for {
			// GET-запрос к серверу
			resp, err := http.Get(server)
			if err == nil {

				// Читаем ответ сервера
				body, _ := io.ReadAll(resp.Body)

				// Разбиваем текст на строки (сообщения)
				lines := strings.Split(strings.TrimSpace(string(body)), "\n")

				// Если появились новые сообщения
				if len(lines) > last_count && lines[0] != "" {
					for i := last_count; i < len(lines); i++ {
						display_chan <- lines[i] // отправляем новое сообщение в канал
					}
					last_count = len(lines)
				}

				resp.Body.Close()
			}

			// Пауза между запросами
			time.Sleep(2 * time.Second)
		}
	}()

	// Запрос имени пользователя
	fmt.Println("Подключено. Введите ваше имя:")
	input_scanner := bufio.NewScanner(os.Stdin)
	input_scanner.Scan()
	user_name := input_scanner.Text()

	// Ввод и отправка сообщений
	fmt.Println("Введите сообщение:")
	for input_scanner.Scan() {
		message := input_scanner.Text()

		// Формируем сообщение с именем
		full_message := "[" + user_name + "]: " + message

		// Отправляем сообщение на сервер POST-запросом
		http.Post(server, "text/plain", strings.NewReader(full_message))
	}
}
