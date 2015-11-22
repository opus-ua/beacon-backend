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
    "strconv"
    "crypto/rand"
    . "github.com/opus-ua/beacon-post"
    . "github.com/opus-ua/beacon-db"
)

const (
    MAX_IMG_BYTES = 1 << 22
)

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
    return imgBytes, nil
}

func ToRespCommentMsg(comment Comment, viewerID int64, db *DBClient) (RespCommentMsg, error) {
    username, err := db.GetUsername(comment.PosterID)
    if err != nil {
        return RespCommentMsg{}, err
    }
    var hearted bool
    if viewerID >= 0 {
        hearted, err = db.HasHearted(comment.ID, uint64(viewerID))
        if err != nil {
            return RespCommentMsg{}, err
        }
    } else {
        hearted = false
    }
    return RespCommentMsg{
        SubmitCommentMsg: SubmitCommentMsg{
            Id:     comment.ID,
            Poster: comment.PosterID,
            Text:   comment.Text,
        },
        RespPostMsg: RespPostMsg{
            Hearts: comment.Hearts,
            Time:   comment.Time.Format(time.UnixDate),
            Username: username,
            Hearted: hearted,
        },
    }, nil
}

func ToRespBeaconMsg(beacon Beacon, viewerID int64, db *DBClient) (RespBeaconMsg, error) {
    username, err := db.GetUsername(beacon.PosterID)
    if err != nil {
        return RespBeaconMsg{}, err
    }
    var hearted bool
    if viewerID >= 0 {
        hearted, err = db.HasHearted(beacon.ID, uint64(viewerID))
        if err != nil {
            return RespBeaconMsg{}, err
        }
    } else {
        hearted = false
    }
    comments := []RespCommentMsg{}
    for _, comment := range beacon.Comments {
        commentMsg, err := ToRespCommentMsg(comment, viewerID, db)
        if err != nil {
            return RespBeaconMsg{}, err
        }
        comments = append(comments, commentMsg)
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
            Username:   username,
            Hearted:    hearted,
        },
        Comments:   comments,
    }, nil
}

func GetAuthenticationInfo(r *http.Request) (uint64, []byte, error) {
    userIDStr, authKeyStr, ok := r.BasicAuth()
    if !ok {
        return 0, []byte{}, errors.New("Could not parse BasicAuth.")
    }
    userIDSigned, err := strconv.ParseInt(userIDStr, 10, 64)
    if err != nil {
        return 0, []byte{}, errors.New("Could not parse user ID as integer.")
    }
    userID := uint64(userIDSigned)
    authKey := []byte(authKeyStr)
    return userID, authKey, nil
}

func Authenticate(r *http.Request, db *DBClient) (uint64, error) {
    userID, authKey, err := GetAuthenticationInfo(r)
    if err != nil {
        return 0, err
    }
    if authed, err := db.UserAuthenticated(userID, authKey); !authed || err != nil {
        return 0, errors.New("Could not authenticate user.")
    }
    return userID, nil
}

func HandlePostBeacon(w http.ResponseWriter, r *http.Request, db *DBClient) {
    ip := r.RemoteAddr
    userID, err := Authenticate(r, db)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, err.Error(), 400)
        return
    }
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
    id, err := db.AddBeacon(&post, userID)
    if err != nil {
        ErrorJSON(w, "Database error.", 500)
        return
    }
    postedBeacon, err := db.GetThread(id)
    respBeaconMsg, err := ToRespBeaconMsg(postedBeacon, int64(userID), db)
    if err != nil {
        ErrorJSON(w, "Database error.", 500)
        return
    }
    respJson, err := json.Marshal(respBeaconMsg)
    if err != nil {
        ErrorJSON(w, "Could not marshal response JSON.", 500)
        return
    }
    io.WriteString(w, string(respJson))
}

func HandleGetBeacon(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    var viewerID int64
    viewerIDu, err := Authenticate(r, db)
    viewerID = int64(viewerIDu)
    if err != nil {
        viewerID = -1
    }
    beacon, err := db.GetThread(id)
    if err != nil {
        ErrorJSON(w, "Could not retrieve post from db.", 404)
        return
    }
    respBeaconMsg, err := ToRespBeaconMsg(beacon, viewerID, db)
    if err != nil {
        ErrorJSON(w, "Database error.", 500)
        return
    }
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
    userID, err := Authenticate(r, db)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, err.Error(), 400)
        return
    }
    err = db.HeartPost(id, userID)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Could not heart post.", 500)
        return;
    }
    w.WriteHeader(200)
}

func HandleUnheartPost(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    userID, err := Authenticate(r, db)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, err.Error(), 400)
        return
    }
    err = db.UnheartPost(id, userID)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Could not unheart post.", 500)
        return;
    }
    w.WriteHeader(200)
}

func HandleFlagPost(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    userID, err := Authenticate(r, db)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, err.Error(), 400)
        return
    }
    err = db.FlagPost(id, userID)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Could not flag post.", 500)
        return;
    }
    w.WriteHeader(200)
}

func GenerateSecret() (string, error) {
    const secretLen = 50
    const characters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" 
    buf := make([]byte, secretLen)
    _, err := rand.Read(buf)
    if err != nil {
        return "", err
    }
    for i := 0; i < secretLen; i++ {
       buf[i] = characters[int(buf[i]) % len(characters)] 
    }
    return string(buf), nil
}

func HandleCreateAccount(w http.ResponseWriter, r *http.Request, googleID []string, db *DBClient) {
    decoder := json.NewDecoder(r.Body)
    var accountReq CreateAccountReqMsg
    decoder.Decode(&accountReq)
    url := "https://www.googleapis.com/oauth2/v3/tokeninfo?id_token=" + accountReq.Token
    res, err := http.Get(url)
    if err != nil {
        log.Printf(err.Error())
        ErrorJSON(w, "Failed to authenticate Google account.", 500)
        return
    }
    if res.StatusCode != 200 {
        log.Printf("Account authentication failure.")
        ErrorJSON(w, "Failed to authenticate Google account.", 500)
        return
    }
    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        log.Printf("Account authentication failure.")
        ErrorJSON(w, "Failed to authenticate Google account.", 500)
        return
    }
    var googleAuth GoogleAuthRespMsg
    err = json.Unmarshal(body, &googleAuth)
    if err != nil {
        log.Printf("Account authentication failure. (json unmarshal)")
        ErrorJSON(w, "Failed to authenticate Google account.", 500)
        return
    }
    if googleAuth.Aud != googleID[0] {
        ErrorJSON(w, "Failed to authenticate Google account", 400)
        return
    }
    secret, err := GenerateSecret()
    if err != nil {
        log.Printf("Failed to generate secret.")
    }
    log.Printf("Successfully authenticated.")
    var id uint64
    if exists, err := db.EmailExists(googleAuth.Email); exists || err != nil {
        id, err = db.GetUserIDByEmail(googleAuth.Email)
        if err != nil {
            log.Printf("Could not find user by email.")
            ErrorJSON(w, "Failed to locate user by email.", 500)
            return
        }
        err = db.SetUserAuthKey(id, []byte(secret))
        if err != nil {
            log.Printf("Could not set user authorization.")
            ErrorJSON(w, "Failed to set user authorization.", 500)
            return
        }
    } else {
        if exists, err := db.UsernameExists(accountReq.Username); exists || err != nil {
            log.Printf("Username '%s' already exists.", accountReq.Username)
            ErrorJSON(w, "Username already exists.", 400)
            return
        }
        id, err = db.CreateUser(accountReq.Username, []byte(secret), googleAuth.Email)
        if err != nil {
            log.Printf("Could not create new user.")
            ErrorJSON(w, "Failed to create new user.", 500)
            return
        }
    }
    respMsg := CreateAccountRespMsg{ID: id, Secret: secret}
    respJson, err := json.Marshal(respMsg)
    if err != nil {
        log.Printf("Could not marshal JSON response.")
        ErrorJSON(w, "Failed to marshal response JSON.", 500)
        return
    }
    io.WriteString(w, string(respJson))
}
