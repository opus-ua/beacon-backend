package beaconpost

import (
    "math"
    "gopkg.in/redis.v3"
    "time"
    "fmt"
    "encoding/binary"
    "bytes"
    "strconv"
    "errors"
)

const (
   REDIS_EXPIRE = 86400 * time.Second
   REDIS_INT_BASE = 36
)

type Geotag struct {
    Latitude    float64
    Longitude   float64
}

func (tag *Geotag) MarshalBinary() ([]byte, error) {
    buf := new(bytes.Buffer)
    err := binary.Write(buf, binary.LittleEndian, tag.Latitude)
    if err != nil {
        return []byte{}, err
    }
    err = binary.Write(buf, binary.LittleEndian, tag.Longitude)
    if err != nil {
        return []byte{}, err
    }
    return buf.Bytes(), nil
}

func (tag *Geotag) UnmarshalBinary(data []byte) error {
    buf := bytes.NewReader(data)
    err := binary.Read(buf, binary.LittleEndian, &tag.Latitude)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.LittleEndian, &tag.Longitude)
    if err != nil {
        return err
    }
    return nil
}

func ToRadians(degrees float64) float64 {
    return degrees * math.Pi / 180
}

// Calculates great circle distance between
// two geotags in *kilometers*
func Distance(p1 Geotag, p2 Geotag) float64 {
    delta := Geotag {
        Latitude:   ToRadians(p2.Latitude - p1.Latitude),
        Longitude:  ToRadians(p2.Longitude - p1.Longitude),
    }
    a := math.Pow(math.Sin(delta.Latitude / 2.0), 2) +
            math.Cos(ToRadians(p1.Latitude)) +
            math.Cos(ToRadians(p2.Latitude)) +
            math.Pow(math.Sin(delta.Longitude / 2.0), 2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0 - a))
    return c * 6371.0
}

func KilometersToMiles(km float64) float64 {
    return 0.621371 * km
}

type BeaconPost struct {
    ID          uint64
    Image       []byte
    Location    Geotag
    PosterID    uint64
    Description string
    Hearts      uint32
    Flags       uint32
    Time        time.Time
    Comments    []Comment
}

type BeaconThumb struct {
    ID          uint64
    Thumb       []byte
    Location    Geotag
    PosterID    uint64
    Description string
    Hearts      uint32
    Time        time.Time
}

type Comment struct {
    ID          uint64
    PosterID    uint64
    BeaconID    uint64
    Text        string
    Hearts      uint32
    Flags       uint32
    Time        time.Time
}

func GetBeacon(id uint64, client *redis.Client) (BeaconPost, error) {
    if post, err := GetBeaconRedis(id, client); err == nil {
        return post, nil
    }
    /*
    if post, err = GetPostGres(id, gresClient); err == nil {
        return post, nil
    }
    */
    return BeaconPost{}, nil
}

func GetBeaconRedis(id uint64, client *redis.Client) (BeaconPost, error) {
    key := fmt.Sprintf("p:%d", id)
    res, err := client.HGetAllMap(key).Result()
    if err != nil {
        return BeaconPost{}, err
    }
    var geotag Geotag
    err = geotag.UnmarshalBinary([]byte(res["loc"]))
    if err != nil {
        return BeaconPost{}, err
    }
    poster, err := strconv.ParseInt(res["poster"], REDIS_INT_BASE, 64)
    posteru := uint64(poster)
    if err != nil {
        return BeaconPost{}, err
    }
    hearts64, err := strconv.ParseInt(res["hearts"], REDIS_INT_BASE, 64)
    if err != nil {
        return BeaconPost{}, err
    }
    hearts := uint32(hearts64)
    flags64, err := strconv.ParseInt(res["flags"], REDIS_INT_BASE, 64)
    if err != nil {
        return BeaconPost{}, err
    }
    flags := uint32(flags64)
    timeText := []byte(res["time"])
    postTime := time.Time{}
    if err = postTime.UnmarshalText(timeText); err != nil {
        return BeaconPost{}, err
    }
    post := BeaconPost{
        ID: id,
        Image: []byte(res["img"]),
        Location: geotag,
        PosterID: posteru,
        Description: res["desc"],
        Hearts: hearts,
        Flags: flags,
        Time: postTime,
    }
    // retrieve comments
    commListKey := fmt.Sprintf("p:%d:c", id)
    comments, err := client.LRange(commListKey, 0, -1).Result()
    for _, commentID := range comments {
        commentKey := fmt.Sprintf("p:%s", commentID)
        commentIDSigned, err := strconv.ParseInt(commentID, REDIS_INT_BASE, 64)
        if err != nil {
            return BeaconPost{}, nil
        }
        commentIDInt := uint64(commentIDSigned)
        commHash, err := client.HGetAllMap(commentKey).Result()
        if err != nil {
            return BeaconPost{}, err
        }
        posterSigned, err := strconv.ParseInt(commHash["poster"], REDIS_INT_BASE, 64)
        if err != nil {
            return BeaconPost{}, err
        }
        poster := uint64(posterSigned)
        parent := id
        hearts64, err = strconv.ParseInt(commHash["hearts"], REDIS_INT_BASE, 64)
        if err != nil {
            return BeaconPost{}, err
        }
        hearts := uint32(hearts64)
        if flags64, err = strconv.ParseInt(commHash["flags"], REDIS_INT_BASE, 64); err != nil {
            return BeaconPost{}, err
        }
        flags := uint32(flags64)
        timeText = []byte(res["time"])
        commentTime := time.Time{}
        if err = commentTime.UnmarshalText(timeText); err != nil {
            return BeaconPost{}, err
        }
        comment := Comment{
            ID: commentIDInt,
            PosterID: poster,
            BeaconID: parent,
            Text: commHash["text"],
            Hearts: hearts,
            Flags: flags,
            Time: commentTime,
        }
        post.Comments = append(post.Comments, comment)
    }
    return post, nil
}

func (post *BeaconPost) Add(client *redis.Client) {
   post.AddRedis(client)
   // post.AddPostGres()
}

func (post *BeaconPost) AddRedis(client *redis.Client) error {
    postID, err := client.Incr("post-count").Result()
    if postID < 0 {
        return errors.New("Retrieved post count was negative.")
    }
    post.ID = uint64(postID)
    if err != nil {
        return err
    }
    key := fmt.Sprintf("p:%d", post.ID)
    locBytes, _ := post.Location.MarshalBinary()
    locString := string(locBytes[:])
    nowb, _ := time.Now().MarshalText()
    now := string(nowb)
    client.HMSet(key, "img", string(post.Image[:]),
                        "loc", locString,
                        "poster", strconv.FormatUint(post.PosterID, REDIS_INT_BASE),
                        "desc", post.Description,
                        "hearts", strconv.FormatUint(uint64(post.Hearts), REDIS_INT_BASE),
                        "flags", strconv.FormatUint(uint64(post.Flags), REDIS_INT_BASE),
                        "time", now,
                        "type", "beacon")
    client.Expire(key, REDIS_EXPIRE)
    return nil
}

func (comment *Comment) Add(client *redis.Client) error {
    comment.AddRedis(client)
    // comment.AddPostGres()
    return nil
}

func (comment *Comment) AddRedis(client *redis.Client) error {
    commentID, err := client.Incr("post-count").Result()
    if commentID < 0 {
        return errors.New("Retrieved post count was negative.")
    }
    comment.ID = uint64(commentID)
    if err != nil {
        return err
    }
    beaconKey := fmt.Sprintf("p:%d", comment.BeaconID)
    IDKey := fmt.Sprintf("p:%d:c", comment.BeaconID)
    client.RPush(IDKey, strconv.FormatUint(comment.ID, REDIS_INT_BASE))
    commKey := fmt.Sprintf("p:%d", comment.ID)
    nowb, _ := time.Now().MarshalText()
    now := string(nowb)
    client.HMSet(commKey, "poster", strconv.FormatUint(comment.PosterID, REDIS_INT_BASE),
                            "parent", strconv.FormatUint(comment.BeaconID, REDIS_INT_BASE),
                            "text", comment.Text,
                            "hearts", strconv.FormatUint(uint64(comment.Hearts), REDIS_INT_BASE),
                            "flags", strconv.FormatUint(uint64(comment.Flags), REDIS_INT_BASE),
                            "time", now,
                            "type", "comment")
    client.Expire(beaconKey, REDIS_EXPIRE)
    client.Expire(IDKey, REDIS_EXPIRE)
    client.Expire(commKey, REDIS_EXPIRE)
    return nil
}
