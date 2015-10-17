package beacondb

import (
    "gopkg.in/redis.v3"
    "time"
    "fmt"
    "strconv"
    "errors"
    . "github.com/opus-ua/beacon-post"
)

const (
   REDIS_EXPIRE = 86400 * time.Second
   REDIS_INT_BASE = 36
)

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

func AddBeacon(post *BeaconPost, client *redis.Client) {
   AddBeaconRedis(post, client)
   // post.AddPostGres()
}

func AddBeaconRedis(post *BeaconPost, client *redis.Client) error {
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

func AddComment(comment *Comment, client *redis.Client) error {
    AddCommentRedis(comment, client)
    // comment.AddPostGres()
    return nil
}

func AddCommentRedis(comment *Comment, client *redis.Client) error {
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
