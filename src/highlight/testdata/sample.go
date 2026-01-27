package main

import (
	"fmt"
	"strings"
)

func main() {
	// This file uses tabs for indentation (Go standard)
	message := "Hello, World!"

	if len(message) > 0 {
		fmt.Println(message)
	}

	// Test with multiple levels of indentation
	for i := 0; i < 3; i++ {
		switch i {
		case 0:
			fmt.Println("First")
		case 1:
			fmt.Println("Second")
		default:
			fmt.Println("Other")
		}
	}

	// Test with comments containing special chars: äöü 🎨
	result := processString("test	data")
	fmt.Println(result)
}

func processString(input string) string {
	// Tab character in string literal above
	return strings.ToUpper(input)
}
