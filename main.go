package main

import (
	"context"
	"log"
)

func main() {
	printVersion()
	checkFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := NewHandler(ctx, botToken, path)

	go h.botUpdate()
	log.Fatal("serve error", h.Serve(addr))
}
