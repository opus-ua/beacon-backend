package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
    "io"
	. "github.com/opus-ua/beacon-rest"
)

var version string = "0.0.0"
var releaseGoogleID string = ""
var debugGoogleID string = ""
var DEFAULT_PORT uint = 8765

var (
	port        uint
	gitHash     string
	showVersion bool
	devMode     bool
)

func init() {
	flag.UintVar(&port, "port", DEFAULT_PORT, "the app will listen on this port")
	flag.UintVar(&port, "p", DEFAULT_PORT, "the app will listen on this port")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&devMode, "dev", false, "start in dev mode")
	flag.Parse()
}

func StartServer(dev bool, testing bool) {
	log.Printf("Listening on port %d.\n", port)
	cores := runtime.NumCPU()
	log.Printf("Core Count: %d", cores)
    versionInfo := VersionInfo{
        Number: version,
        Hash: gitHash,
        DevMode: devMode,
    }
    server := NewBeaconServer(dev, testing, versionInfo, []string{releaseGoogleID, debugGoogleID})
	err := server.Start(port)
	if err != nil {
		if port == DEFAULT_PORT {
			log.Printf("Is an instance of Beacon already running?\n")
		}
		log.Fatal(err.Error())
	}
}

func PrintVersion() {
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Git Hash: %s\n", gitHash)
}

func main() {
	logFile, err := os.OpenFile("/var/log/beacon", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Could not open log file '/var/log/beacon'. %s", err.Error())
		os.Exit(1)
	}
	logWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(logWriter)
	if showVersion {
		PrintVersion()
	} else {
		StartServer(devMode, false)
	}
}
