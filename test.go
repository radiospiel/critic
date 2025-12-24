package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Hello, World!")
	fmt.Println("Current time:", time.Now())
	greet("Critic")
}

func greet(name string) {
	fmt.Printf("Welcome to %s!\n", name)
}
