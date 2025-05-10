package main

import (
	"context"
	"flag"
	"fmt"
	"log"
)

var BuildVersion = "dev"

func main() {
	conf := flag.String("config", "config.json", "path to config file or a http(s) url")
	version := flag.Bool("version", false, "print version and exit")
	help := flag.Bool("help", false, "print help and exit")
	flag.Parse()
	if *help {
		flag.Usage()
		return
	}
	if *version {
		fmt.Println(BuildVersion)
		return
	}
	config, err := LoadConfig(*conf)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startHTTPServer(ctx, config)
}
