package beaconrest

type SubmitPostMsg struct {
    Id           uint64 `json:"id"`
    Poster       uint64 `json:"userid"`
    Text         string `json:"text"`
}

type RespPostMsg struct {
    Hearts      uint32 `json:"hearts"`
    Time        string `json:"time"`
    Username    string `json:"username"`
    Hearted     bool   `json:"hearted"`
}

type LocationMsg struct {
    Latitude     float64 `json:"latitude"`
    Longitude    float64 `json:"longitude"`
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

type CreateAccountReqMsg struct {
    Username string `json:"username"`
    Token   string `json:"token"`
}

type CreateAccountRespMsg struct {
    ID uint64 `json:"id"`
    Secret string `json:"secret"`
}

type GoogleAuthRespMsg struct {
    Iss string `json:"iss"`
    Sub string `json:"sub"`
    Azp string `json:"azp"`
    Email string `json:"email"`
    AtHash string `json:"at_hash"`
    EmailVerified string `json:"email_verified"`
    Aud string `json:"aud"`
    Iat string `json:"iat"`
    Exp string `json:"exp"`
}

type LocalSearchMsg struct {
    Latitude float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Radius float64 `json:"radius"`
}

type LocalSearchRespMsg struct {
    Beacons []SubmitBeaconMsg `json:"beacons"`
}

type PostCommentMsg struct {
    BeaconID    uint64 `json:"beaconid"`
    Text        string `json:"text"`
}
