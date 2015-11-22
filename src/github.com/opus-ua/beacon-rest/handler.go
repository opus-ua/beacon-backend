package beaconrest

import (
    "mime/multipart"
    "net/textproto"
    "mime"
    "net/http"
    "log"
    "encoding/json"
    "strings"
    "io/ioutil"
    "time"
    "io"
    "bytes"
    "strconv"
    "crypto/rand"
    "errors"
    . "github.com/opus-ua/beacon-post"
    . "github.com/opus-ua/beacon-db"
)

const (
    MAX_IMG_BYTES = 1 << 22
)

func ParsePostBeaconJson(w http.ResponseWriter, part *multipart.Part, ip string) (Beacon, error) {
    jsonBytes, err := ioutil.ReadAll(part)
    if err != nil {
        return Beacon{}, WriteErrorResp(w, "Unable to read json body.", JsonError)
    }
    var beaconMsg SubmitBeaconMsg
    err = json.Unmarshal(jsonBytes, &beaconMsg)
    if err != nil {
        return Beacon{}, WriteErrorResp(w, "Unable to parse json body.", JsonError)
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
        return []byte{}, WriteErrorResp(w, "Unable to read image.", ProtocolError)
    }
    return imgBytes, nil
}

func ToRespCommentMsg(w http.ResponseWriter, comment Comment, viewerID int64, db *DBClient) (RespCommentMsg, error) {
    username, err := db.GetUsername(comment.PosterID)
    if err != nil {
        return RespCommentMsg{}, WriteErrorResp(w, err.Error(), DatabaseError)
    }
    var hearted bool
    if viewerID >= 0 {
        hearted, err = db.HasHearted(comment.ID, uint64(viewerID))
        if err != nil {
            return RespCommentMsg{}, WriteErrorResp(w, err.Error(), DatabaseError)
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

func ToRespBeaconMsg(w http.ResponseWriter, beacon Beacon, viewerID int64, db *DBClient) (RespBeaconMsg, error) {
    username, err := db.GetUsername(beacon.PosterID)
    if err != nil {
        return RespBeaconMsg{}, WriteErrorResp(w, err.Error(), DatabaseError)
    }
    var hearted bool
    if viewerID >= 0 {
        hearted, err = db.HasHearted(beacon.ID, uint64(viewerID))
        if err != nil {
            return RespBeaconMsg{}, WriteErrorResp(w, err.Error(), DatabaseError)
        }
    } else {
        hearted = false
    }
    comments := []RespCommentMsg{}
    for _, comment := range beacon.Comments {
        commentMsg, err := ToRespCommentMsg(w, comment, viewerID, db)
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

func GetAuthenticationInfo(w http.ResponseWriter, r *http.Request) (int64, []byte, error) {
    userIDStr, authKeyStr, ok := r.BasicAuth()
    if !ok {
        return 0, []byte{}, errors.New("Unable to parse BasicAuth.")
    }
    userID, err := strconv.ParseInt(userIDStr, 10, 64)
    if err != nil {
        return 0, []byte{}, errors.New("Unable to read user ID.")
    }
    authKey := []byte(authKeyStr)
    return userID, authKey, nil
}

func Authenticate(w http.ResponseWriter, r *http.Request, db *DBClient) (uint64, error) {
    userIDSigned, authKey, err := GetAuthenticationInfo(w, r)
    if err != nil {
        return 0, err
    }
    userID := uint64(userIDSigned)
    if authed, err := db.UserAuthenticated(userID, authKey); !authed || err != nil {
        return 0, WriteErrorResp(w, err.Error(), DatabaseError)
    }
    return userID, nil
}

func OptionalAuthenticate(w http.ResponseWriter, r *http.Request, db *DBClient) (int64, error) {
    userID, authKey, err := GetAuthenticationInfo(w, r)
    if err != nil {
        return -1, nil
    }
    if authed, err := db.UserAuthenticated(uint64(userID), authKey); !authed || err != nil {
        return 0, WriteErrorResp(w, err.Error(), DatabaseError)
    }
    return userID, nil
}

func HandlePostBeacon(w http.ResponseWriter, r *http.Request, db *DBClient) {
    ip := r.RemoteAddr
    userID, err := Authenticate(w, r, db)
    if err != nil {
        return
    }
    mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
    if err != nil {
        WriteErrorResp(w, "Content-Type not found.", ProtocolError)
        return
    }
    if !strings.HasPrefix(mediaType, "multipart/") {
        WriteErrorResp(w, "Received non-multipart message.", ProtocolError)
        return
    }
    multiReader := multipart.NewReader(r.Body, params["boundary"])
    jsonPart, err := multiReader.NextPart()
    if err != nil {
        WriteErrorResp(w, "Didn't receive enough message parts.", ProtocolError)
        return
    }
    post, err := ParsePostBeaconJson(w, jsonPart, ip)
    if err != nil {
        return
    }
    imgPart, err := multiReader.NextPart()
    if err != nil {
        log.Printf(err.Error())
        WriteErrorResp(w, "No image found in message.", ProtocolError)
        return
    }
    img, err := GetPostBeaconImg(w, imgPart, ip)
    if err != nil {
        return
    }
    post.Image = img
    id, err := db.AddBeacon(&post, userID)
    if err != nil {
        WriteErrorResp(w, err.Error(), DatabaseError)
        return
    }
    postedBeacon, err := db.GetThread(id)
    if err != nil {
        WriteErrorResp(w, err.Error(), DatabaseError)
        return
    }
    respBeaconMsg, err := ToRespBeaconMsg(w, postedBeacon, int64(userID), db)
    if err != nil {
        return
    }
    respJson, err := json.Marshal(respBeaconMsg)
    if err != nil {
        WriteErrorResp(w, err.Error(), JsonError)
        return
    }
    io.WriteString(w, string(respJson))
}

func HandleGetBeacon(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    var viewerID int64
    viewerID, err := OptionalAuthenticate(w, r, db)
    beacon, err := db.GetThread(id)
    if err != nil {
        WriteErrorResp(w, err.Error(), DatabaseError)
        return
    }
    respBeaconMsg, err := ToRespBeaconMsg(w, beacon, viewerID, db)
    if err != nil {
        return
    }
    respJson, err := json.Marshal(respBeaconMsg)
    respBody := &bytes.Buffer{}
    partWriter := multipart.NewWriter(respBody)
    jsonHeader := textproto.MIMEHeader{}
    jsonHeader.Add("Content-Type", "application/json")
    jsonWriter, err := partWriter.CreatePart(jsonHeader)
    if err != nil {
        WriteErrorResp(w, err.Error(), ServerError)
        return
    }
    jsonWriter.Write(respJson)
    imgHeader := textproto.MIMEHeader{}
    imgHeader.Add("Content-Type", "img/jpeg")
    imgWriter, err := partWriter.CreatePart(imgHeader)
    if err != nil {
        WriteErrorResp(w, err.Error(), ServerError)
        return
    }
    imgWriter.Write(beacon.Image)
    partWriter.Close()
    w.Header().Add("Content-Type", partWriter.FormDataContentType())
    w.Write(respBody.Bytes())
}

func HandleHeartPost(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    userID, err := Authenticate(w, r, db)
    if err != nil {
        return
    }
    err = db.HeartPost(id, userID)
    if err != nil {
        WriteErrorResp(w, err.Error(), DatabaseError)
        return
    }
    w.WriteHeader(200)
}

func HandleUnheartPost(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    userID, err := Authenticate(w, r, db)
    if err != nil {
        return
    }
    err = db.UnheartPost(id, userID)
    if err != nil {
        WriteErrorResp(w, err.Error(), DatabaseError)
        return
    }
    w.WriteHeader(200)
}

func HandleFlagPost(w http.ResponseWriter, r *http.Request, id uint64, db *DBClient) {
    userID, err := Authenticate(w, r, db)
    if err != nil {
        return
    }
    err = db.FlagPost(id, userID)
    if err != nil {
        WriteErrorResp(w, err.Error(), DatabaseError)
        return;
    }
    w.WriteHeader(200)
}

func GenerateSecret(w http.ResponseWriter) (string, error) {
    const secretLen = 50
    const characters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" 
    buf := make([]byte, secretLen)
    _, err := rand.Read(buf)
    if err != nil {
        return "", WriteErrorResp(w, "Failed to generate secret.", ServerError)
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
        WriteErrorResp(w, err.Error(), ExternalServiceError)
        return
    }
    if res.StatusCode != 200 {
        WriteErrorResp(w, "Failed to authenticate Google account.", ExternalServiceError)
        return
    }
    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        WriteErrorResp(w, "Failed to authenticate Google account.", ServerError)
        return
    }
    var googleAuth GoogleAuthRespMsg
    err = json.Unmarshal(body, &googleAuth)
    if err != nil {
        WriteErrorResp(w, "Failed to authenticate Google account.", JsonError)
        return
    }
    inKeys := false
    for _, key := range googleID {
        if key == googleAuth.Aud {
            inKeys = true
        }
    }
    if !inKeys {
        WriteErrorResp(w, "Failed to authenticate Google account", ProtocolError)
        return
    }
    secret, err := GenerateSecret(w)
    if err != nil {
        return
    }
    var id uint64
    if exists, err := db.EmailExists(googleAuth.Email); exists || err != nil {
        id, err = db.GetUserIDByEmail(googleAuth.Email)
        if err != nil {
            WriteErrorResp(w, err.Error(), NoAccountFound)
            return
        }
        err = db.SetUserAuthKey(id, []byte(secret))
        if err != nil {
            WriteErrorResp(w, err.Error(), ServerError)
            return
        }
    } else {
        if exists, err := db.UsernameExists(accountReq.Username); exists || err != nil {
            WriteErrorResp(w, "Username exists.", UsernameExists)
            return
        }
        id, err = db.CreateUser(accountReq.Username, []byte(secret), googleAuth.Email)
        if err != nil {
            WriteErrorResp(w, err.Error(), DatabaseError)
            return
        }
    }
    respMsg := CreateAccountRespMsg{ID: id, Secret: secret}
    respJson, err := json.Marshal(respMsg)
    if err != nil {
        WriteErrorResp(w, err.Error(), ServerError)
        return
    }
    io.WriteString(w, string(respJson))
}
