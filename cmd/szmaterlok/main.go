package main

import "log"

func run() error {
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal("szmaterlok:", err.Error())
	}
}
