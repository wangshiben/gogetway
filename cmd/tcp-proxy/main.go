package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/wangshiben/gogetway/getwayServer"
	"github.com/wangshiben/gogetway/tcpPlayback"
)

func main() {
	listenAddr := flag.String("listen", "", "TCP listen address, or observed service address in passive mirror mode")
	flag.StringVar(listenAddr, "l", "", "shorthand for -listen")
	forwardAddr := flag.String("forward", "", "TCP forward address, replay target address, or mirror target address")
	flag.StringVar(forwardAddr, "f", "", "shorthand for -forward")
	outputFile := flag.String("log", "", "record file path, or replay input file when -replay is set")
	flag.StringVar(outputFile, "o", "", "shorthand for -log")
	replayMode := flag.Bool("replay", false, "replay packets from -log to -forward")
	flag.BoolVar(replayMode, "r", false, "shorthand for -replay")
	device := flag.String("device", "", "capture device for passive mirror mode; omit to scan all devices")
	flag.StringVar(device, "d", "", "shorthand for -device")
	imageOnly := flag.Bool("image-only", false, "capture mirrored traffic only, without forwarding to -forward")
	flag.BoolVar(imageOnly, "io", false, "shorthand for -image-only")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *replayMode {
		if *forwardAddr == "" || *outputFile == "" {
			flag.Usage()
			os.Exit(2)
		}
		fmt.Printf("replaying packets from %s to %s\n", *outputFile, *forwardAddr)
		count, err := tcpPlayback.ReplayFileToTarget(*outputFile, *forwardAddr, false, nil)
		if err != nil {
			log.Fatalf("replay failed: %v", err)
		}
		fmt.Printf("replayed %d packets from %s to %s\n", count, *outputFile, *forwardAddr)
		return
	}

	writer, writeFunc, closeOutput, err := openOutput(*outputFile)
	if err != nil {
		log.Fatalf("open output failed: %v", err)
	}
	defer closeOutput()

	if *imageOnly || *device != "" {
		if *listenAddr == "" {
			flag.Usage()
			os.Exit(2)
		}
		observedPort, err := parseObservedPort(*listenAddr)
		if err != nil {
			log.Fatalf("parse observed port failed: %v", err)
		}
		targetAddr := *forwardAddr
		if *imageOnly {
			targetAddr = ""
		}
		if targetAddr == "" && !*imageOnly {
			flag.Usage()
			os.Exit(2)
		}
		source, err := newObservedSource(*device, observedPort)
		if err != nil {
			log.Fatalf("create live observed source failed: %v", err)
		}
		mirror := getwayServer.NewGopacketTrafficMirror(getwayServer.GopacketMirrorConfig{
			ObservedPort: observedPort,
			TargetAddr:   targetAddr,
			Writer:       writer,
			WriteFunc:    writeFunc,
		})
		if *imageOnly {
			if *outputFile == "" {
				fmt.Printf("mirroring TCP traffic to observed service %s on all available devices without forwarding\n", *listenAddr)
			} else {
				fmt.Printf("mirroring TCP traffic to observed service %s on all available devices and writing proto packets to %s\n", *listenAddr, *outputFile)
			}
		} else if *outputFile == "" {
			fmt.Printf("mirroring TCP traffic to observed service %s on device %s and forwarding to %s\n", *listenAddr, sourceLabel(*device), targetAddr)
		} else {
			fmt.Printf("mirroring TCP traffic to observed service %s on device %s, forwarding to %s, writing proto packets to %s\n", *listenAddr, sourceLabel(*device), targetAddr, *outputFile)
		}
		err = mirror.Start(ctx, source)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Fatalf("traffic mirror failed: %v", err)
		}
		return
	}

	if *listenAddr == "" || *forwardAddr == "" {
		flag.Usage()
		os.Exit(2)
	}

	server := getwayServer.NewSimpleTCPServerWithWriterAndFunc(*forwardAddr, *listenAddr, getwayServer.TCPType, writer, writeFunc)
	if *outputFile == "" {
		fmt.Printf("listening on %s, forwarding to %s\n", *listenAddr, *forwardAddr)
	} else {
		fmt.Printf("listening on %s, forwarding to %s, writing proto packets to %s\n", *listenAddr, *forwardAddr, *outputFile)
	}
	server.StartListen()
}

func openOutput(outputFile string) (io.Writer, getwayServer.WriteFunc, func(), error) {
	if outputFile == "" {
		writer := io.Discard
		return writer, func(data []byte, _ context.Context) (int, error) {
			return writer.Write(data)
		}, func() {}, nil
	}
	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, nil, err
	}
	writer := io.Writer(file)
	var writeLock sync.Mutex
	writeFunc := func(data []byte, _ context.Context) (int, error) {
		writeLock.Lock()
		defer writeLock.Unlock()
		return writer.Write(data)
	}
	return writer, writeFunc, func() {
		_ = file.Close()
	}, nil
}

func newObservedSource(device string, observedPort uint16) (getwayServer.ObservedPacketSource, error) {
	if device != "" {
		return newLiveObservedTCPPortSource(device, observedPort)
	}
	return newAutoObservedTCPPortSource(observedPort)
}

func sourceLabel(device string) string {
	if device == "" {
		return "all available devices"
	}
	return device
}

func parseObservedPort(listenAddr string) (uint16, error) {
	portText := listenAddr
	if strings.Contains(listenAddr, ":") {
		_, port, err := net.SplitHostPort(listenAddr)
		if err != nil {
			return 0, err
		}
		portText = port
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return 0, err
	}
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port: %d", port)
	}
	return uint16(port), nil
}
