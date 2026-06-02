package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app, err := CreateApp(*config)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	app.Discord.AddHandlerOnce(app.HandleReady)
	app.Discord.AddHandler(app.HandleMessageCreate)

	if err := app.Discord.Open(); err != nil {
		log.Fatalf("open discord session: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-sig

	log.Println("Shutting down...")
	if err := app.Discord.Close(); err != nil {
		log.Printf("error closing discord session: %v", err)
	}
}
