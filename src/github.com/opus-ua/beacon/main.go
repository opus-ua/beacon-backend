package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/opus-ua/beacon-rest"
	"gopkg.in/redis.v3"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
)

var version string = "0.0.0"
var DEFAULT_PORT uint = 8765

var (
	port        uint
	gitHash     string
	showVersion bool
	devMode     bool
)

type VersionInfo struct {
	Number string `json:"version"`
	Hash   string `json:"hash"`
}

type JSONError struct {
	Text string `json:"error"`
}

func ErrorJSON(w http.ResponseWriter, s string, code int) {
	r, _ := json.Marshal(JSONError{Text: s})
	http.Error(w, string(r), code)
	log.Printf(s)
}

func init() {
	flag.UintVar(&port, "port", DEFAULT_PORT, "the app will listen on this port")
	flag.UintVar(&port, "p", DEFAULT_PORT, "the app will listen on this port")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&devMode, "dev", false, "start in dev mode")

	flag.Parse()
}

func HandleVersion(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("Received version request from %s.\n", r.RemoteAddr)
	log.Printf(msg)
	switch r.Method {
	case "GET":
		versionJSON, err := json.Marshal(VersionInfo{Number: version, Hash: gitHash})
		if err != nil {
			ErrorJSON(w, "Could not retrieve version number.", http.StatusInternalServerError)
			return
		}
		io.WriteString(w, string(versionJSON))

	default:
		ErrorJSON(w, fmt.Sprintf("Unsupported method %s.", r.Method), http.StatusBadRequest)
	}
}

func StartServer(dev bool) {
	log.Printf("Listening on port %d.\n", port)
	cores := runtime.NumCPU()
	log.Printf("Core Count: %d", cores)

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	if dev {
        log.Printf("Starting in dev mode.")
		if err := client.Select(11).Err(); err != nil {
			log.Printf("Could not select unused dev database.\n")
			os.Exit(1)
		}
	}

	http.HandleFunc("/version", HandleVersion)
	http.HandleFunc("/beacon", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received beacon post request.\n")
		HandlePostBeacon(w, r, client)
	})
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
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
	if showVersion {
		PrintVersion()
	} else {
		StartServer(devMode)
	}
}
