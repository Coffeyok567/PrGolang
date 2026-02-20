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

	fmt.Print("Введите ник: ")
	scanner.Scan()
	name := scanner.Text()

	http.Post(server, "text/plain", strings.NewReader("register="+name))

	// ===== ЧАТ: ПРИЁМ =====
	go func() {
		last := 0
		for {
			resp, err := http.Get(server)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				lines := strings.Split(strings.TrimSpace(string(body)), "\n")
				if len(lines) > last {
					for i := last; i < len(lines); i++ {
						fmt.Println("[CHAT]", lines[i])
					}
					last = len(lines)
				}
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// ===== ЧАТ: ВВОД =====
	go func() {
		for scanner.Scan() {
			text := scanner.Text()
			http.Post(server, "text/plain",
				strings.NewReader("chat=["+name+"]: "+text))
		}
	}()

	// ===== PVP =====
	for {
		resp, _ := http.Get(server)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		state := string(body)

		switch state {
		case "WAIT":
			time.Sleep(2 * time.Second)

		case "ATTACK":
			fmt.Print("АТАКА (head/body/legs): ")
			scanner.Scan()
			a := scanner.Text()
			http.Post(server, "text/plain",
				strings.NewReader("attack="+name+":"+a))

		case "DEFENSE":
			fmt.Print("ЗАЩИТА (head/body/legs): ")
			scanner.Scan()
			d := scanner.Text()
			http.Post(server, "text/plain",
				strings.NewReader("defense="+name+":"+d))
		}
	}
}
