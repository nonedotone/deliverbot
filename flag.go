package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	botToken string
	path     string
	admin    int64
	addr     string
	url      string
	version  bool
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	flag.StringVar(&botToken, "token", "", "token of telegram bot")
	flag.StringVar(&path, "path", "./db", "path/to/db")
	flag.Int64Var(&admin, "admin", 0, "admin of bot(if not set(0),everyone can use it)")
	flag.StringVar(&addr, "addr", ":8000", "http serve addr")
	flag.StringVar(&url, "example-url", "example.com/deliverbot", "example url")
	flag.BoolVar(&version, "version", false, "version of deliverbot")
	flag.Parse()
}

func printVersion() {
	if version {
		fmt.Println(Version())
		os.Exit(0)
	}
}

func checkFlags() {
	if len(botToken) == 0 {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
}
