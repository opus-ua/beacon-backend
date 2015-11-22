package beaconrest

import (
    "encoding/json"
    "net/http"
    "fmt"
    "log"
    "runtime"
    "path"
)

type JSONError struct {
    Code int `json:"code"`
	Msg string `json:"error"`
}

func (e JSONError) Error() string {
    return fmt.Sprintf("Error %d: \"%s\"", e.Code, e.Msg)
}

type ErrResp struct {
    HttpCode int
    HttpMsg string
}

const (
    DatabaseError = 30
    ProtocolError = 31
    ServerError = 32
    ExternalServiceError = 33
    JsonError = 34
    NoAccountFound = 40
    UsernameExists = 41
    UnspecifiedError = 99
)

var errorCodes map[int]ErrResp

func init() {
    errorCodes = map[int]ErrResp{
        30: ErrResp{HttpCode: 500, HttpMsg: "Database error."},
        31: ErrResp{HttpCode: 500, HttpMsg: "Server error."},
        32: ErrResp{HttpCode: 400, HttpMsg: "Protocol error."},
        33: ErrResp{HttpCode: 400, HttpMsg: "External service error."},
        34: ErrResp{HttpCode: 400, HttpMsg: "Json error."},
        40: ErrResp{HttpCode: 400, HttpMsg: "No account found."},
        41: ErrResp{HttpCode: 400, HttpMsg: "Username already exists."},
        99: ErrResp{HttpCode: 500, HttpMsg: "Unspecified error."},
    }
}

func WriteErrorResp(w http.ResponseWriter, debugMsg string, errCode int) error {
    err, ok := errorCodes[errCode]
    if !ok {
        err = errorCodes[99]
    }
    _, file, line, _ := runtime.Caller(1)
    errMsg := fmt.Sprintf("%s:%d: %s", path.Base(file), line, debugMsg)
    jsonObj := JSONError{Code: errCode, Msg: errMsg}
    log.Printf(jsonObj.Error())
    jsonErr, _ := json.Marshal(JSONError{Code: errCode, Msg: err.HttpMsg})
    http.Error(w, string(jsonErr), err.HttpCode)
    return jsonObj
}
