package beaconpost

import (
    "gopkg.in/redis.v3"
    "testing"
    "fmt"
    "os"
    "reflect"
)

var client *redis.Client = nil

func TestMain(m *testing.M) {
    client = redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        Password: "",
        DB: 0,
    })
    if err := client.Select(11).Err(); err != nil {
        fmt.Printf("Could not select unused database.\n")
        os.Exit(1)
    }
    client.FlushDb()
    res := m.Run()
    client.FlushDb()
    os.Exit(res)
}

func RedisExpect(res *redis.StringCmd, expected string, t *testing.T) {
    if res.Err() != nil {
        t.Fatalf(res.Err().Error())
    }
    if resVal, _ := res.Result(); resVal != expected {
        t.Fatalf("'%s' != '%s'", resVal, expected)
    }
}

func TestAddBeacon(t *testing.T) {
    p := BeaconPost{
        Image: []byte("abcde"),
        Location: Geotag{Latitude: 45.0, Longitude: 45.0},
        PosterID: 54321,
        Description: "Go go Redis!",
        Hearts: 5,
        Flags: 1,
    }
    p.Add(client)
    key := fmt.Sprintf("p:%d", p.ID)
    RedisExpect(client.HGet(key, "img"), "abcde", t)
    RedisExpect(client.HGet(key, "loc"), "\x00\x00\x00\x00\x00\x80F@\x00\x00\x00\x00\x00\x80F@", t)
    RedisExpect(client.HGet(key, "poster"), "15wx", t)
    RedisExpect(client.HGet(key, "desc"), "Go go Redis!", t)
    RedisExpect(client.HGet(key, "hearts"), "5", t)
    RedisExpect(client.HGet(key, "flags"), "1", t)
    RedisExpect(client.HGet(key, "type"), "beacon", t)
}

func TestAddComment(t *testing.T) {
    postIDSigned, _ := client.Get("post-count").Int64()
    postID := uint64(postIDSigned)
    commentA := Comment{
        PosterID: 54321,
        BeaconID: postID,
        Text: "For real. This is stuff.",
        Hearts: 1,
        Flags: 0,
    }
    commentB := Comment{
        PosterID: 626,
        BeaconID: postID,
        Text: "Reed sucks.",
        Hearts: 0,
        Flags: 27,
   }
   commentA.Add(client)
   commentB.Add(client)
   commentListKey := fmt.Sprintf("p:%d:c", postID)
   res, err := client.LRange(commentListKey, 0, -1).Result()
   if err != nil {
        t.Fatalf(err.Error())
   }
   if !reflect.DeepEqual(res, []string{"2", "3"}) {
        t.Fatalf("Comment list was not correct.")
   }
   key := fmt.Sprintf("p:%d", commentA.ID)
   RedisExpect(client.HGet(key, "poster"), "15wx", t)
   RedisExpect(client.HGet(key, "parent"), "1", t)
   RedisExpect(client.HGet(key, "text"), "For real. This is stuff.", t)
   RedisExpect(client.HGet(key, "hearts"), "1", t)
   RedisExpect(client.HGet(key, "flags"), "0", t)
   RedisExpect(client.HGet(key, "type"), "comment", t)
   key = fmt.Sprintf("p:%d", commentB.ID)
   RedisExpect(client.HGet(key, "poster"), "he", t)
   RedisExpect(client.HGet(key, "parent"), "1", t)
   RedisExpect(client.HGet(key, "text"), "Reed sucks.", t)
   RedisExpect(client.HGet(key, "hearts"), "0", t)
   RedisExpect(client.HGet(key, "flags"), "r", t)
   RedisExpect(client.HGet(key, "type"), "comment", t)
}
