package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/AndrezHP/musync/cmd"
)

func main() {
	startLogging()
	start := time.Now()

	cmd.Run()

	end := time.Now()
	elapsed := end.Sub(start)
	log.Println("Elapsed:", time.Duration.Milliseconds(elapsed))
}

func startLogging() {
	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)
}
