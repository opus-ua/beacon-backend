package beacon

import (
    "testing"
    "net/http"
    "io/ioutil"
    "strings"
    "time"
)

func TestGetVersion(t *testing.T) {
    go StartServer()
    time.Sleep(50 * time.Millisecond)
    resp, err := http.Get("http://localhost:8765/version")
    if err != nil {
        t.Fatalf("Could not connect to beacon backend.")
    }
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        t.Fatalf("Could not parse response body.")
    }
    if !strings.Contains(string(body),"version") {
        t.Fatalf("Response body did not contain 'version'.")
    }

    t.Log(string(body))
}
