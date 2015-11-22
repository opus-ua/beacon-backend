package beaconrest

import (
	"encoding/json"
    "net/http"
    "fmt"
    "strings"
    "strconv"
	"io"
	. "github.com/opus-ua/beacon-db"
)

type BeaconServer struct {
    db *DBClient
    mux *http.ServeMux
    authCodes []string
    version VersionInfo
}

func NewBeaconServer(dev bool, testing bool, version VersionInfo, auth []string) *BeaconServer {
    bs := &BeaconServer{
        db: NewDB(dev, testing),
        mux: http.DefaultServeMux,
        authCodes: auth,
        version: version,
    }
    bs.HandleVersion("/version")
    bs.HandleAuth("/createaccount", "POST", HandleCreateAccount)
    bs.HandlePost("/beacon", HandlePostBeacon)
    bs.HandleIntParam("/beacon/", "GET", HandleGetBeacon)
    bs.HandleIntParam("/heart/", "POST", HandleHeartPost)
    bs.HandleIntParam("/unheart/", "POST", HandleUnheartPost)
    bs.HandleIntParam("/flag/", "POST", HandleFlagPost)
    return bs
}

type BeaconHandler func(http.ResponseWriter, *http.Request, *DBClient)
type IntParamBeaconHandler func(http.ResponseWriter, *http.Request, uint64, *DBClient)
type AuthBeaconHandler func(http.ResponseWriter, *http.Request, []string, *DBClient)

func (bm *BeaconServer) Start(port uint) error {
    loggingHandler := NewApacheLoggingHandler(bm.mux)
    server := &http.Server{
        Addr: fmt.Sprintf(":%d", port),
        Handler: loggingHandler,
    }
    return server.ListenAndServe()
}

func (bm *BeaconServer) TestingMode() error {
    return bm.db.TestingTable()
}

func (bm *BeaconServer) HandleMethod(uri string, method string, handler BeaconHandler) {
    bm.mux.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
        if r.Method != method {
            msg := fmt.Sprintf("Only method %s supported.", method)
            WriteErrorResp(w, msg, ProtocolError)
            return
        }
        handler(w, r, bm.db)
    })
}

func (bm *BeaconServer) HandleGet(uri string, handler BeaconHandler) {
    bm.HandleMethod(uri, "GET", handler)
}

func (bm *BeaconServer) HandlePost(uri string, handler BeaconHandler) {
    bm.HandleMethod(uri, "POST", handler)
}

func (bm *BeaconServer) HandleIntParam(uri string, method string, handler IntParamBeaconHandler) {
    bm.HandleMethod(uri, method, func(w http.ResponseWriter, r *http.Request, db *DBClient) {
        splitURI := strings.Split(r.RequestURI, "/")
        if len(splitURI) < 3 {
            WriteErrorResp(w, "Could not parse uri parameter.", ProtocolError)
            return
        }
        intStr := splitURI[2]
        intSigned, err := strconv.ParseInt(intStr, 10, 64)
        if err != nil {
            WriteErrorResp(w, "Could not parse uri parameter.", ProtocolError)
            return
        }
        handler(w, r, uint64(intSigned), db)
    })
}

func (bm *BeaconServer) HandleAuth(uri string, method string, handler AuthBeaconHandler) {
    bm.HandleMethod(uri, method, func(w http.ResponseWriter, r *http.Request, db *DBClient) {
        handler(w, r, bm.authCodes, db)
    })
}

type VersionInfo struct {
	Number  string `json:"version"`
	Hash    string `json:"hash"`
	DevMode bool   `json:"dev-mode"`
}

func (bm *BeaconServer) HandleVersion(uri string) {
    bm.HandleGet(uri, func(w http.ResponseWriter, r *http.Request, db *DBClient) {
        versionJSON, err := json.Marshal(bm.version)
        if err != nil {
            WriteErrorResp(w, "Could not retrieve version number.", ServerError)
            return
        }
        io.WriteString(w, string(versionJSON))
    })
}
