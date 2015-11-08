package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/opus-ua/beacon-dummy"
	. "github.com/opus-ua/beacon-rest"
	"gopkg.in/redis.v3"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
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
	Number  string `json:"version"`
	Hash    string `json:"hash"`
	DevMode bool   `json:"dev-mode"`
}

func init() {
	flag.UintVar(&port, "port", DEFAULT_PORT, "the app will listen on this port")
	flag.UintVar(&port, "p", DEFAULT_PORT, "the app will listen on this port")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&devMode, "dev", false, "start in dev mode")

	flag.Parse()
}

func HandleVersion(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		versionJSON, err := json.Marshal(VersionInfo{Number: version, Hash: gitHash, DevMode: devMode})
		if err != nil {
			ErrorJSON(w, "Could not retrieve version number.", http.StatusInternalServerError)
			return
		}
		io.WriteString(w, string(versionJSON))

	default:
		ErrorJSON(w, fmt.Sprintf("Unsupported method %s.", r.Method), http.StatusBadRequest)
	}
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		handler.ServeHTTP(w, r)
	})
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
		client.FlushDb()
		AddDummy(client)
	}

	mux := http.DefaultServeMux
	mux.HandleFunc("/version", HandleVersion)
	mux.HandleFunc("/beacon", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			HandlePostBeacon(w, r, client)
		} else {
			ErrorJSON(w, "Only method POST supported.", 400)
		}
	})
	mux.HandleFunc("/beacon/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			splitURI := strings.Split(r.RequestURI, "/")
			if len(splitURI) < 3 {
				ErrorJSON(w, "Could not parse post id.", http.StatusBadRequest)
				return
			}
			idStr := splitURI[2]
			idSigned, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				log.Fatal(err.Error())
				ErrorJSON(w, "Could not parse post id.", http.StatusBadRequest)
				return
			}
			id := uint64(idSigned)
			HandleGetBeacon(w, r, id, client)
		} else {
			ErrorJSON(w, "Only method GET supported.", 400)
		}
	})
	mux.HandleFunc("/heart/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			splitURI := strings.Split(r.RequestURI, "/")
			if len(splitURI) < 3 {
				ErrorJSON(w, "Could not parse post id.", http.StatusBadRequest)
				return
			}
			idStr := splitURI[2]
			idSigned, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				log.Fatal(err.Error())
				ErrorJSON(w, "Could not parse post id.", http.StatusBadRequest)
				return
			}
			id := uint64(idSigned)
			HandleHeartPost(w, r, id, client)
		} else {
			ErrorJSON(w, "Only method POST supported.", 400)
		}
	})
	mux.HandleFunc("/flag/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			splitURI := strings.Split(r.RequestURI, "/")
			if len(splitURI) < 3 {
				ErrorJSON(w, "Could not parse post id.", http.StatusBadRequest)
				return
			}
			idStr := splitURI[2]
			idSigned, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				log.Fatal(err.Error())
				ErrorJSON(w, "Could not parse post id.", http.StatusBadRequest)
				return
			}
			id := uint64(idSigned)
			HandleFlagPost(w, r, id, client)
		} else {
			ErrorJSON(w, "Only method POST supported.", 400)
		}
	})

	loggingHandler := NewApacheLoggingHandler(mux)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: loggingHandler,
	}
	err := server.ListenAndServe()
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
		StartServer(devMode)
	}
}
