package main

import (
    "io"
    "net/http"
    "runtime"
    "fmt"
)

func hello(w http.ResponseWriter, r *http.Request) {
    io.WriteString(w, "Hello world!")
}

func main() {
    cores := runtime.NumCPU()
    fmt.Printf("Core Count: %d", cores)

    http.HandleFunc("/", hello)
    http.ListenAndServe(":9999", nil)
}
