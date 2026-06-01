package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config := LoadConfig()
	app := CreateApp(*config)

	defer app.Discord.Close()

	app.Discord.AddHandlerOnce(app.HandleReady)
	app.Discord.AddHandler(app.HandleMessageCreate)

	app.Discord.Open()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-sig

	log.Println("Shutting down...")
	os.Exit(0)
}
