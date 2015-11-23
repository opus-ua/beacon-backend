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
    return fmt.Sprintf("%s \"%s\"", errorCodes[e.Code].HttpMsg, e.Msg)
}

type ErrResp struct {
    HttpCode int
    HttpMsg string
}

const (
    ProtocolError = 31
    JsonError = 32
    DatabaseError = 40
    ServerError = 41
    ExternalServiceError = 42
    NoAccountFound = 50
    UsernameExists = 51
    UnspecifiedError = 99
)

var errorCodes map[int]ErrResp

func init() {
    errorCodes = map[int]ErrResp{
        31: ErrResp{HttpCode: 400, HttpMsg: "Protocol error."},
        32: ErrResp{HttpCode: 400, HttpMsg: "Json error."},
        40: ErrResp{HttpCode: 500, HttpMsg: "Database error."},
        41: ErrResp{HttpCode: 500, HttpMsg: "Server error."},
        42: ErrResp{HttpCode: 400, HttpMsg: "External service error."},
        50: ErrResp{HttpCode: 400, HttpMsg: "No account found."},
        51: ErrResp{HttpCode: 400, HttpMsg: "Username already exists."},
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
