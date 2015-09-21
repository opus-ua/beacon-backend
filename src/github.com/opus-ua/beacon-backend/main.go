package main

import (
    "io"
    "net/http"
    "runtime"
    "fmt"
    "flag"
    "log"
    "encoding/json"
)

var version string = "0.0.0"

var (
    port        uint
    showVersion bool
)

type VersionInfo struct {
    Number  string  `json:"version"`
}

type JSONError struct {
    Text    string  `json:"error"`
}

func ErrorJSON(w http.ResponseWriter, s string, code int) {
    r, _ := json.Marshal(JSONError{Text: s})
    http.Error(w, string(r), code)
    log.Printf(s)
}

func init() {
    flag.UintVar(&port, "port", 8765, "the app will listen on this port")
    flag.UintVar(&port, "p", 8765, "the app will listen on this port")
    flag.BoolVar(&showVersion, "version", false, "show version information")

    flag.Parse()
}

func HandleVersion(w http.ResponseWriter, r *http.Request) {
    if r.Method == "GET" {
        versionJSON, err := json.Marshal(VersionInfo{Number: version})
        if err != nil {
            ErrorJSON(w, "Could not retrieve version number.", http.StatusInternalServerError)
        } else {
            io.WriteString(w, string(versionJSON))
        }
    } else {
        ErrorJSON(w, fmt.Sprintf("Unsupported method %s.", r.Method), http.StatusBadRequest)
    }
}

func StartServer() {
    log.Printf("Listening on port %d.\n", port)
    cores := runtime.NumCPU()
    log.Printf("Core Count: %d", cores)

    http.HandleFunc("/version", HandleVersion)
    http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func PrintVersion() {
    fmt.Printf("Version %s.\n", version)
}

func main() {
    if showVersion {
        PrintVersion()
    } else {
        StartServer()
    }
}
