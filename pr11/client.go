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
	server := "https://musical-carnival-v6qv54v7j4443vjg-8080.app.github.dev/"
	scanner := bufio.NewScanner(os.Stdin)

	// ===== ввод ника =====
	fmt.Print("Введите ник: ")
	scanner.Scan()
	name := scanner.Text()

	http.Post(server+"/pvp", "text/plain",
		strings.NewReader("register="+name))

	// ===== ЧАТ: получение =====
	go func() {
		last := 0
		for {
			resp, err := http.Get(server + "/chat")
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				lines := strings.Split(strings.TrimSpace(string(body)), "\n")

				if len(lines) > last && lines[0] != "" {
					for i := last; i < len(lines); i++ {
						fmt.Println(lines[i])
					}
					last = len(lines)
				}
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
	}()

	// ===== ЧАТ: ввод =====
	go func() {
		for scanner.Scan() {
			text := scanner.Text()
			http.Post(server+"/chat", "text/plain",
				strings.NewReader("["+name+"]: "+text))
		}
	}()

	// ===== PVP =====
	for {
		resp, _ := http.Get(server + "/pvp")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		state := string(body)

		switch state {
		case "WAIT":
			time.Sleep(1 * time.Second)

		case "ATTACK":
			fmt.Print("АТАКА (head/body/legs): ")
			scanner.Scan()
			a := scanner.Text()
			http.Post(server+"/pvp", "text/plain",
				strings.NewReader("attack="+name+":"+a))

		case "DEFENSE":
			fmt.Print("ЗАЩИТА (head/body/legs): ")
			scanner.Scan()
			d := scanner.Text()
			http.Post(server+"/pvp", "text/plain",
				strings.NewReader("defense="+name+":"+d))
		}
	}
}

