package beaconrest

import (
    "mime/multipart"
    "http"
    "errors"
    "log"
    "io/ioutil"
    "encoding/json"
    "fmt"
    "gopkg.in/redis.v3"
    . "github.com/opus-ua/beaconpost"
)

const (
    MAX_IMG_BYTES 1 << 22
)

type PostBeaconMsg struct {
    Poster       uint64 `json:"user"`
    Latitude     float64 `json:"long"`
    Longitude    float64 `json:"lat"`
    Description  string `json:"desc"`
}

type PostCommentMsg struct {
    Poster      int `json:"user"`
    Text        string `json:"text"`
}

func ParsePostBeaconJson(w http.ResponseWriter, r *http.Request) (BeaconPost, error) {
    ip := r.Header().Get("x-forwarded-for")
    jsonFile, jsonHeader, err := r.FormFile("json")
    if err != nil {
        msg := fmt.Sprintf("Unable to read message from %s: %s", ip, err.Error())
        http.Error(w, "", 500)
        return BeaconPost{}, errors.New(msg)
    }
    defer jsonBody.Close()
    jsonBody, err := ioutil.ReadAll(jsonFile)
    if err != nil {
        msg := fmt.Sprintf("Unable to read JSON in message from %s: %s", ip, err.Error())
        http.Error(w, "", 500)
        return BeaconPost{}, errors.New(msg)
    }
    var beaconMsg PostBeaconMsg
    err = json.Unmarshal(jsonBody, &beaconMsg)
    if err != nil {
        msg := fmt.Sprintf("Unable to parse JSON in message from %s: %s", ip, err.Error())
        http.Error(w, "{\"error\": \"Malformed JSON.\"}", 400)
        return BeaconPost{}, errors.New(msg)
    }
    post := BeaconPost{
        ID: beaconMsg.Poster,
        Location: Geotag{Latitude: beaconMsg.Latitude, Longitude: beaconMsg.Longitude},
        Description: beaconMsg.Description,
        Hearts: 0,
        Flags: 0,
    }
    return post, nil
}

func GetPostBeaconImg(w http.ResponseWriter, r *http.Request) ([]byte, error) {
    ip := r.Header().Get("x-forwarded-for")
    imgFile, imgHeader, err := r.FormFile("img")
    defer imgFile.Close()
    img, err := ioutil.ReadAll(imgFile)
    if err != nil {
        msg := fmt.Sprintf("Unable to read image from %s: %s", ip, err.Error())
        http.Error(w, "", 500)
        return []byte{}, errors.New(msg)
    }
    return img, nil
}

func HandlePostBeacon(w http.ResponseWriter, r *http.Request, client *redis.Client) {
    ip := r.Header().Get("x-forwarded-for")
    r.ParseMultiform(MAX_IMG_BYTES)
    post, err := ParsePostBeaconJson(w, r)
    if err != nil {
        log.Printf(err.Error())
        return
    }
    img, err := GetPostBeaconImg(w, r)
    post.Image = img
    err := AddBeacon(&post, client)
    if err != nil {
        msg := fmt.Sprintf("Database error for connection to %s: %s", ip, err.Error())
        http.Error(w, "{\"error\": \"Database error.\"}", 500)
        return
    }
}
