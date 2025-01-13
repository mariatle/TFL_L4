package main

import (
	"bufio"
	"fmt"
	"lab4/parser"
	"os"
)

func main() {()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Введите регулярку: ")

		if !scanner.Scan() {
			break
		}
		input := scanner.Text()

		tokens, err := parser.Tokenize(input)
		if err != nil {
			fmt.Println("Ошибка:", err)
			continue
		}
		for _, token := range tokens {
			fmt.Println(token)
		}
		node, err := parser.ConvertToAST(tokens)
		if err != nil {
			fmt.Println("Ошибка:", err)
			continue
		}
		fmt.Println(node)
		cfg := parser.ConvertToCFG(node)
		for key, rules := range cfg {
			for _, rule := range rules {
				fmt.Printf("%s -> ", key)
				for _, nt := range rule {
					fmt.Print(nt, " ")
				}
				fmt.Println()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Ошибка при чтении ввода: %v\n", err)
	}
}
