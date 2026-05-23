package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/wangshiben/gogetway/getwayServer"
)

func main() {
	listenAddr := flag.String("listen", "", "TCP listen address, for example :9000 or 127.0.0.1:9000")
	flag.StringVar(listenAddr, "l", "", "shorthand for -listen")
	forwardAddr := flag.String("forward", "", "TCP forward address, for example 127.0.0.1:8080")
	flag.StringVar(forwardAddr, "f", "", "shorthand for -forward")
	outputFile := flag.String("log", "", "record file path")
	flag.StringVar(outputFile, "o", "", "shorthand for -log")
	flag.Parse()

	if *listenAddr == "" || *forwardAddr == "" {
		flag.Usage()
		os.Exit(2)
	}

	writer := io.Discard
	writeFunc := func(data []byte, _ context.Context) (int, error) {
		return writer.Write(data)
	}

	if *outputFile != "" {
		file, err := os.OpenFile(*outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("open output file failed: %v", err)
		}
		defer file.Close()

		var writeLock sync.Mutex
		writer = file
		writeFunc = func(data []byte, _ context.Context) (int, error) {
			writeLock.Lock()
			defer writeLock.Unlock()
			return writer.Write(data)
		}
	}

	server := getwayServer.NewSimpleTCPServerWithWriterAndFunc(*forwardAddr, *listenAddr, getwayServer.TCPType, writer, writeFunc)

	if *outputFile == "" {
		fmt.Printf("listening on %s, forwarding to %s\n", *listenAddr, *forwardAddr)
	} else {
		fmt.Printf("listening on %s, forwarding to %s, writing proto packets to %s\n", *listenAddr, *forwardAddr, *outputFile)
	}
	server.StartListen()
}
