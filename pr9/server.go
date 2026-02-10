package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

var chat_history []string
var history_mutex sync.Mutex
var server_output = make(chan string, 10)

func main() {
	go func() {
		for log_msg := range server_output {
			fmt.Println(log_msg)
		}
	}()

	go func() {
		server_scanner := bufio.NewScanner(os.Stdin)
		for server_scanner.Scan() {
			server_text := server_scanner.Text()
			full_server_msg :=  server_text
			
			history_mutex.Lock()
			chat_history = append(chat_history, full_server_msg)
			history_mutex.Unlock()
			
			server_output <- "Вы: " + server_text
		}
	}()

	http.HandleFunc("/", func(response_writer http.ResponseWriter, request_data *http.Request) {
		if request_data.Method == http.MethodPost {
			body_bytes, _ := io.ReadAll(request_data.Body)
			client_msg := string(body_bytes)
			
			history_mutex.Lock()
			chat_history = append(chat_history, client_msg)
			history_mutex.Unlock()
			
			server_output <- "Клиент: " + client_msg
			fmt.Fprint(response_writer, "получено")
		} else {
			history_mutex.Lock()
			for _, single_msg := range chat_history {
				fmt.Fprintln(response_writer, single_msg)
			}
			history_mutex.Unlock()
		}
	})

	server_output <- "Сервер запущен"
	http.ListenAndServe(":8080", nil)
}