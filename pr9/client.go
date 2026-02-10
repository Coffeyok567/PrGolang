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

var display_chan = make(chan string, 5)

func main() {
	server := "http://localhost:8080"  //вот тут большой шаурма (ссылку на развернутый сервер на гит хабе)
	go func() {
		for text_to_print := range display_chan {
			fmt.Println(text_to_print)
		}
	}()
  
	go func() {
		last_count := 0
		for {
			resp, err := http.Get(server)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				lines := strings.Split(strings.TrimSpace(string(body)), "\n")
				
				if len(lines) > last_count && lines[0] != "" {
					for i := last_count; i < len(lines); i++ {
						display_chan <- lines[i]
					}
					last_count = len(lines)
				}
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}
	}()

	fmt.Println("Подключено. Введите ваше имя:")
	input_scanner := bufio.NewScanner(os.Stdin)
	input_scanner.Scan()
	user_name := input_scanner.Text()

	fmt.Println("Введите сообщение:")
	for input_scanner.Scan() {
		message := input_scanner.Text()
		full_message := "[" + user_name + "]: " + message
		http.Post(server, "text/plain", strings.NewReader(full_message))
	}

}
