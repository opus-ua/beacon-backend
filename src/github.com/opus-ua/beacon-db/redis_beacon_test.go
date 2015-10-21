package beacondb

import (
	"fmt"
	. "github.com/opus-ua/beacon-post"
	"gopkg.in/redis.v3"
	"os"
	"reflect"
	"strconv"
	"testing"
)

var client *redis.Client = nil

func TestMain(m *testing.M) {
	client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
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

func RedisNotNil(res *redis.StringCmd, t *testing.T) {
	if res.Err() != nil {
		t.Fatalf(res.Err().Error())
	}
}

var p Beacon = Beacon{
	Image:       []byte("abcde"),
	Location:    Geotag{Latitude: 45.0, Longitude: 45.0},
	PosterID:    54321,
	Description: "Go go Redis!",
	Hearts:      5,
	Flags:       1,
}

func TestAddBeacon(t *testing.T) {
    err := AddBeacon(&p, client)
    if err != nil {
        t.Fatalf(err.Error())
    }
	key := fmt.Sprintf("p:%d", p.ID)
	RedisExpect(client.HGet(key, "img"), "abcde", t)
	RedisExpect(client.HGet(key, "loc"), "\x00\x00\x00\x00\x00\x80F@\x00\x00\x00\x00\x00\x80F@", t)
	RedisExpect(client.HGet(key, "poster"), "15wx", t)
	RedisExpect(client.HGet(key, "desc"), "Go go Redis!", t)
	RedisExpect(client.HGet(key, "hearts"), "5", t)
	RedisExpect(client.HGet(key, "flags"), "1", t)
	RedisExpect(client.HGet(key, "type"), "beacon", t)
	RedisNotNil(client.HGet(key, "time"), t)
}

var commentA Comment = Comment{
	PosterID: 54321,
	BeaconID: 1,
	Text:     "For real. This is stuff.",
	Hearts:   1,
	Flags:    0,
}

var commentB Comment = Comment{
	PosterID: 626,
	BeaconID: 1,
	Text:     "Reed sucks.",
	Hearts:   0,
	Flags:    27,
}

func TestAddComment(t *testing.T) {
	AddComment(&commentA, client)
	AddComment(&commentB, client)
	commentListKey := "p:1:c"
	res, err := client.LRange(commentListKey, 0, -1).Result()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !reflect.DeepEqual(res, []string{"2", "3"}) {
		fmt.Printf("Expected: ['2', '3']\nRetrieved: %v", res)
		t.Fatalf("Comment list was not correct.")
	}
	key := fmt.Sprintf("p:%d", commentA.ID)
	RedisExpect(client.HGet(key, "poster"), "15wx", t)
	RedisExpect(client.HGet(key, "parent"), "1", t)
	RedisExpect(client.HGet(key, "text"), "For real. This is stuff.", t)
	RedisExpect(client.HGet(key, "hearts"), "1", t)
	RedisExpect(client.HGet(key, "flags"), "0", t)
	RedisExpect(client.HGet(key, "type"), "comment", t)
	RedisNotNil(client.HGet(key, "time"), t)
	key = fmt.Sprintf("p:%d", commentB.ID)
	RedisExpect(client.HGet(key, "poster"), "he", t)
	RedisExpect(client.HGet(key, "parent"), "1", t)
	RedisExpect(client.HGet(key, "text"), "Reed sucks.", t)
	RedisExpect(client.HGet(key, "hearts"), "0", t)
	RedisExpect(client.HGet(key, "flags"), "r", t)
	RedisExpect(client.HGet(key, "type"), "comment", t)
	RedisNotNil(client.HGet(key, "time"), t)
}

func TestGetBeacon(t *testing.T) {
	post, err := GetBeaconRedis(1, client)
	if err != nil {
		t.Fatalf(err.Error())
	}
	commentA.Time = post.Comments[0].Time
	commentB.Time = post.Comments[1].Time
	p.Time = post.Time
	p.Comments = []Comment{commentA, commentB}
	if !reflect.DeepEqual(p, post) {
		fmt.Printf("Stored:\n%v\n", p)
		fmt.Printf("Retrieved:\n%v\n", post)
		t.Fatalf("Retrieved beacon not same as stored beacon.")
	}
}

func BenchmarkAddBeacon(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p.Description = strconv.Itoa(i)
		AddBeacon(&p, client)
	}
}

func BenchmarkRetrieveBeacon(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetBeaconRedis(1, client)
	}
}
