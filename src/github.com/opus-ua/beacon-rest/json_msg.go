package beaconrest

import (
    "mime/multipart"
    "mime"
    "net/http"
    "errors"
    "log"
    "encoding/json"
    "fmt"
    "strings"
    "io/ioutil"
    "gopkg.in/redis.v3"
    "time"
    "io"
    . "github.com/opus-ua/beacon-post"
    . "github.com/opus-ua/beacon-db"
)

const (
    MAX_IMG_BYTES = 1 << 22
)

type SubmitPostMsg struct {
    Id           uint64 `json:"id"`
    Poster       uint64 `json:"user"`
    Text         string `json:"text"`
}

type RespPostMsg struct {
    Hearts      uint32 `json:"hearts"`
    Time        string `json:"time"`
}

type LocationMsg struct {
    Latitude     float64 `json:"long"`
    Longitude    float64 `json:"lat"`
}

type SubmitBeaconMsg struct {
    SubmitPostMsg
    LocationMsg
}

type SubmitCommentMsg SubmitPostMsg

type RespCommentMsg struct {
    SubmitCommentMsg
    RespPostMsg
}

type RespBeaconMsg struct {
    SubmitBeaconMsg
    RespPostMsg
    Comments    []RespCommentMsg `json:"comments"`
}

func ParsePostBeaconJson(w http.ResponseWriter, part *multipart.Part, ip string) (Beacon, error) {
    jsonBytes, err := ioutil.ReadAll(part)
    if err != nil {
        msg := fmt.Sprintf("Unable to read content of json body from %s: %s", ip, err.Error())
        return Beacon{}, errors.New(msg)
    }
    var beaconMsg SubmitBeaconMsg
    log.Printf("Received json: %s (length %d)", string(jsonBytes), len(jsonBytes))
    err = json.Unmarshal(jsonBytes, &beaconMsg)
    if err != nil {
        msg := fmt.Sprintf("Unable to parse JSON in message from %s: %s", ip, err.Error())
        http.Error(w, "{\"error\": \"Malformed JSON.\"}", 400)
        return Beacon{}, errors.New(msg)
    }
    post := Beacon{
        ID: beaconMsg.Poster,
        Location: Geotag{Latitude: beaconMsg.Latitude, Longitude: beaconMsg.Longitude},
        Description: beaconMsg.Text,
        Hearts: 0,
        Flags: 0,
    }
    return post, nil
}

func GetPostBeaconImg(w http.ResponseWriter, part *multipart.Part, ip string) ([]byte, error) {
    imgBytes, err := ioutil.ReadAll(part)
    if err != nil {
        msg := fmt.Sprintf("Unable to read image from %s: %s", ip, err.Error())
        http.Error(w, "", 500)
        return []byte{}, errors.New(msg)
    }
    return imgBytes, nil
}

func ToRespCommentMsg(comment Comment) RespCommentMsg {
    return RespCommentMsg{
        SubmitCommentMsg: SubmitCommentMsg{
            Id:     comment.ID,
            Poster: comment.PosterID,
            Text:   comment.Text,
        },
        RespPostMsg: RespPostMsg{
            Hearts: comment.Hearts,
            Time:   comment.Time.Format(time.UnixDate),
        },
    }
}

func ToRespBeaconMsg(beacon Beacon) RespBeaconMsg {
    comments := []RespCommentMsg{}
    for _, comment := range beacon.Comments {
        comments = append(comments, ToRespCommentMsg(comment))
    }
    return RespBeaconMsg{
        SubmitBeaconMsg: SubmitBeaconMsg{
            SubmitPostMsg: SubmitPostMsg{
                Id:         beacon.ID,
                Poster:     beacon.PosterID,
                Text:       beacon.Description,
            },
            LocationMsg: LocationMsg{
                Latitude:   beacon.Location.Latitude,
                Longitude:  beacon.Location.Longitude,
            },
        },
        RespPostMsg: RespPostMsg{
            Hearts:     beacon.Hearts,
            Time:       beacon.Time.Format(time.UnixDate),
        },
        Comments:   comments,
    }
}

func HandlePostBeacon(w http.ResponseWriter, r *http.Request, client *redis.Client) {
    ip := r.RemoteAddr
    mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
    if err != nil {
        log.Printf(err.Error())
        return
    }
    if !strings.HasPrefix(mediaType, "multipart/") {
        log.Printf("Received non-multipart message.\n")
        return
    }
    multiReader := multipart.NewReader(r.Body, params["boundary"])
    jsonPart, err := multiReader.NextPart()
    if err != nil {
        log.Printf(err.Error())
        return
    }
    post, err := ParsePostBeaconJson(w, jsonPart, ip)
    if err != nil {
        log.Printf(err.Error())
        return
    }
    imgPart, err := multiReader.NextPart()
    img, err := GetPostBeaconImg(w, imgPart, ip)
    if err != nil {
        log.Printf(err.Error())
        return
    }
    post.Image = img
    id, err := AddBeacon(&post, client)
    if err != nil {
        msg := fmt.Sprintf("Database error for connection to %s: %s", ip, err.Error())
        http.Error(w, "{\"error\": \"Database error.\"}", 500)
        log.Printf(msg)
        return
    }
    postedBeacon, err := GetBeacon(id, client)
    respBeaconMsg := ToRespBeaconMsg(postedBeacon)
    respJson, err := json.Marshal(respBeaconMsg)
    if err != nil {
        msg := fmt.Sprintf("Unable to marshal response json.")
        log.Printf(msg)
        return 
    }
    io.WriteString(w, string(respJson))
}
