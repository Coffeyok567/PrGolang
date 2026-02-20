package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

// История чата (сообщения сервера и клиентов)
var chat_history []string

// Мьютекс для безопасного доступа к истории из разных горутин
var history_mutex sync.Mutex

// Канал для вывода сообщений сервера в консоль
var server_output = make(chan string, 10)

func main() {

	// Горутина для вывода сообщений из канала в консоль
	go func() {
		for log_msg := range server_output {
			fmt.Println(log_msg)
		}
	}()

	// Горутина для чтения сообщений из консоли сервера
	go func() {
		server_scanner := bufio.NewScanner(os.Stdin)
		for server_scanner.Scan() {
			server_text := server_scanner.Text()
			full_server_msg := server_text

			// Добавляем сообщение сервера в историю
			history_mutex.Lock()
			chat_history = append(chat_history, full_server_msg)
			history_mutex.Unlock()

			// Отправляем сообщение в канал вывода
			server_output <- "Вы: " + server_text
		}
	}()

	// HTTP-обработчик для корневого пути "/"
	http.HandleFunc("/", func(response_writer http.ResponseWriter, request_data *http.Request) {

		// Если клиент отправляет POST-запрос
		if request_data.Method == http.MethodPost {

			// Читаем тело запроса
			body_bytes, _ := io.ReadAll(request_data.Body)
			client_msg := string(body_bytes)

			// Добавляем сообщение клиента в историю
			history_mutex.Lock()
			chat_history = append(chat_history, client_msg)
			history_mutex.Unlock()

			// Выводим сообщение клиента в консоль сервера
			server_output <- "Клиент: " + client_msg

			// Ответ клиенту
			fmt.Fprint(response_writer, "получено")

		} else {
			// Если GET-запрос — отправляем всю историю чата
			history_mutex.Lock()
			for _, single_msg := range chat_history {
				fmt.Fprintln(response_writer, single_msg)
			}
			history_mutex.Unlock()
		}
	})

	// Сообщение о запуске сервера
	server_output <- "Сервер запущен на http://localhost:8080"
	server_output <- "Ожидание подключений..."
	server_output <- "Введите сообщение для отправки всем клиентам:"

	// Запуск HTTP-сервера на порту 8080
	http.ListenAndServe(":8080", nil)
}
