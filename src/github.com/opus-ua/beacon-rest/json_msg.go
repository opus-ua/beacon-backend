package beaconrest

import (
    "mime/multipart"
    "net/textproto"
    "mime"
    "net/http"
    "errors"
    "log"
    "encoding/json"
    "fmt"
    "strings"
    "io/ioutil"
    "time"
    "io"
    "bytes"
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
    Latitude     float64 `json:"longitude"`
    Longitude    float64 `json:"latitude"`
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

type JSONError struct {
	Text string `json:"error"`
}

func ErrorJSON(w http.ResponseWriter, s string, code int) {
	r, _ := json.Marshal(JSONError{Text: s})
	http.Error(w, string(r), code)
	log.Printf(s)
}

func ParsePostBeaconJson(w http.ResponseWriter, part *multipart.Part, ip string) (Beacon, error) {
    jsonBytes, err := ioutil.ReadAll(part)
    if err != nil {
        msg := fmt.Sprintf("Unable to read content of json body from %s: %s", ip, err.Error())
        return Beacon{}, errors.New(msg)
    }
    var beaconMsg SubmitBeaconMsg
    err = json.Unmarshal(jsonBytes, &beaconMsg)
    if err != nil {
        msg := fmt.Sprintf("Unable to parse JSON in message from %s: %s", ip, err.Error())
        ErrorJSON(w, "Received malformed json.", 400)
        return Beacon{}, errors.New(msg)
    }
    post := Beacon{
        PosterID: beaconMsg.Poster,
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
        ErrorJSON(w, "Unable to read submitted image.", 500)
        return []byte{}, errors.New(msg)
    }
    return imgBytes, nil }

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

func HandlePostBeacon(w http.ResponseWriter, r *http.Request, db *DBClient) {
    ip := r.RemoteAddr
    mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Content-Type not present.", 400)
        return
    }
    if !strings.HasPrefix(mediaType, "multipart/") {
        ErrorJSON(w, "Received non multi-part message.", 400)
        return
    }
    multiReader := multipart.NewReader(r.Body, params["boundary"])
    jsonPart, err := multiReader.NextPart()
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Received too few parts in message.", 400)
        return
    }
    post, err := ParsePostBeaconJson(w, jsonPart, ip)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Received malformed JSON.", 400)
        return
    }
    imgPart, err := multiReader.NextPart()
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "No image present in message.", 400)
        return
    }
    img, err := GetPostBeaconImg(w, imgPart, ip)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Could not read image in message.", 400)
        return
    }
    post.Image = img
    id, err := db.AddBeacon(&post)
    if err != nil {
        ErrorJSON(w, "Database error.", 500)
        return
    }
    postedBeacon, err := db.GetThread(id)
    respBeaconMsg := ToRespBeaconMsg(postedBeacon)
    respJson, err := json.Marshal(respBeaconMsg)
    if err != nil {
        ErrorJSON(w, "Could not marshal response JSON.", 500)
        return
    }
    io.WriteString(w, string(respJson))
}

func HandleGetBeacon(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    beacon, err := db.GetThread(id)
    if err != nil {
        ErrorJSON(w, "Could not retrieve post from db.", 404)
        return
    }
    respBeaconMsg := ToRespBeaconMsg(beacon)
    respJson, err := json.Marshal(respBeaconMsg)
    respBody := &bytes.Buffer{}
    partWriter := multipart.NewWriter(respBody)
    jsonHeader := textproto.MIMEHeader{}
    jsonHeader.Add("Content-Type", "application/json")
    jsonWriter, err := partWriter.CreatePart(jsonHeader)
    if err != nil {
        ErrorJSON(w, "Could not write response.", 500)
        return
    }
    jsonWriter.Write(respJson)
    imgHeader := textproto.MIMEHeader{}
    imgHeader.Add("Content-Type", "img/jpeg")
    imgWriter, err := partWriter.CreatePart(imgHeader)
    if err != nil {
        ErrorJSON(w, "Could not write response.", 500)
        return
    }
    imgWriter.Write(beacon.Image)
    partWriter.Close()
    w.Header().Add("Content-Type", partWriter.FormDataContentType())
    w.Write(respBody.Bytes())
}

func HandleHeartPost(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    err := db.HeartPost(id)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Could not heart post.", 500)
    }
    w.WriteHeader(200)
}

func HandleFlagPost(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    err := db.FlagPost(id)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Could not flag post.", 500)
    }
    w.WriteHeader(200)
}
