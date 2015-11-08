package beacondb

import (
	"encoding"
	"errors"
	"fmt"
	. "github.com/opus-ua/beacon-post"
	"gopkg.in/redis.v3"
	"strconv"
	"time"
)

const (
	REDIS_EXPIRE   = 86400 * time.Second
	REDIS_INT_BASE = 36
)

func GetRedisPostKey(id uint64) string {
	return fmt.Sprintf("p:%d", id)
}

func GetRedisCommentListKey(id uint64) string {
	return fmt.Sprintf("%s:c", GetRedisPostKey(id))
}

func GetBeacon(id uint64, client *redis.Client) (Beacon, error) {
	return GetBeaconRedis(id, client)
	/*
	   if post, err = GetPostGres(id, gresClient); err == nil {
	       return post, nil
	   }
	*/
}

func RedisParseUInt64(res string, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}
	valSigned, err := strconv.ParseInt(res, REDIS_INT_BASE, 64)
	return uint64(valSigned), err
}

func RedisParseUInt32(res string, err error) (uint32, error) {
	if err != nil {
		return 0, err
	}
	val64, err := strconv.ParseInt(res, REDIS_INT_BASE, 64)
	return uint32(val64), err
}

func RedisParseTime(res string, err error) (time.Time, error) {
	if err != nil {
		return time.Now(), err
	}
	return time.Parse(time.UnixDate, res)
}

func RedisParseText(res string, obj encoding.TextUnmarshaler, err error) error {
	if err != nil {
		return err
	}
	err = obj.UnmarshalText([]byte(res))
	return err
}

func RedisParseBinary(res string, obj encoding.BinaryUnmarshaler, err error) error {
	if err != nil {
		return err
	}
	err = obj.UnmarshalBinary([]byte(res))
	return err
}

func GetBeaconOnlyRedis(id uint64, client *redis.Client) (Beacon, error) {
	key := GetRedisPostKey(id)
	res, err := client.HGetAllMap(key).Result()
	if err != nil {
		return Beacon{}, err
	}
	if len(res) == 0 {
		return Beacon{}, errors.New("Beacon not found in db.")
	}
	var geotag Geotag
	err = RedisParseBinary(res["loc"], &geotag, err)
	timePosted, err := RedisParseTime(res["time"], err)
	poster, err := RedisParseUInt64(res["poster"], err)
	hearts, err := RedisParseUInt32(res["hearts"], err)
	flags, err := RedisParseUInt32(res["flags"], err)
	if err != nil {
		return Beacon{}, err
	}
	post := Beacon{
		ID:          id,
		Image:       []byte(res["img"]),
		Location:    geotag,
		PosterID:    poster,
		Description: res["desc"],
		Hearts:      hearts,
		Flags:       flags,
		Time:        timePosted,
	}
	return post, nil
}

func GetCommentRedis(id uint64, parent uint64, client *redis.Client) (Comment, error) {
	commentKey := GetRedisPostKey(id)
	commHash, err := client.HGetAllMap(commentKey).Result()
	commentTime, err := RedisParseTime(commHash["time"], err)
	poster, err := RedisParseUInt64(commHash["poster"], err)
	hearts, err := RedisParseUInt32(commHash["hearts"], err)
	flags, err := RedisParseUInt32(commHash["flags"], err)
	if err != nil {
		return Comment{}, err
	}
	comment := Comment{
		ID:       id,
		PosterID: poster,
		BeaconID: parent,
		Text:     commHash["text"],
		Hearts:   hearts,
		Flags:    flags,
		Time:     commentTime,
	}
	return comment, nil
}

func GetRedisCommentList(id uint64, client *redis.Client) ([]uint64, error) {
	key := GetRedisCommentListKey(id)
	strList, err := client.LRange(key, 0, -1).Result()
	if err != nil {
		return []uint64{}, err
	}
	intList := []uint64{}
	for _, str := range strList {
		commID, err := RedisParseUInt64(str, nil)
		if err != nil {
			return intList, err
		}
		intList = append(intList, commID)
	}
	return intList, nil
}

func GetBeaconRedis(id uint64, client *redis.Client) (Beacon, error) {
	post, err := GetBeaconOnlyRedis(id, client)
	if err != nil {
		return Beacon{}, err
	}
	comments, err := GetRedisCommentList(id, client)
	if err != nil {
		return Beacon{}, err
	}
	for _, commentID := range comments {
		comment, err := GetCommentRedis(commentID, id, client)
		if err != nil {
			return Beacon{}, err
		}
		post.Comments = append(post.Comments, comment)
	}
	return post, nil
}

func AddBeacon(post *Beacon, client *redis.Client) (uint64, error) {
	id, err := AddBeaconRedis(post, client)
	// post.AddPostGres()
	return id, err
}

func AddBeaconRedis(post *Beacon, client *redis.Client) (uint64, error) {
	postID, err := client.Incr("post-count").Result()
	if postID < 0 {
		return 0, errors.New("Retrieved post count was negative.")
	}
	post.ID = uint64(postID)
	if err != nil {
		return 0, err
	}
	key := GetRedisPostKey(post.ID)
	locBytes, _ := post.Location.MarshalBinary()
	locString := string(locBytes[:])
	now := time.Now().Format(time.UnixDate)
	client.HMSet(key, "img", string(post.Image[:]),
		"loc", locString,
		"poster", strconv.FormatUint(post.PosterID, REDIS_INT_BASE),
		"desc", post.Description,
		"hearts", strconv.FormatUint(uint64(post.Hearts), REDIS_INT_BASE),
		"flags", strconv.FormatUint(uint64(post.Flags), REDIS_INT_BASE),
		"time", now,
		"type", "beacon")
	client.Expire(key, REDIS_EXPIRE)
	return post.ID, nil
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
	beaconKey := GetRedisPostKey(comment.BeaconID)
	IDKey := GetRedisCommentListKey(comment.BeaconID)
	client.RPush(IDKey, strconv.FormatUint(comment.ID, REDIS_INT_BASE))
	commKey := GetRedisPostKey(comment.ID)
	now := time.Now().Format(time.UnixDate)
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

func HeartPost(postID uint64, client *redis.Client) error {
	return HeartPostRedis(postID, client)
}

func HeartPostRedis(postID uint64, client *redis.Client) error {
	key := GetRedisPostKey(postID)
	_, err := client.HIncrBy(key, "hearts", 1).Result()
	if err != nil {
		return err
	}
	client.Expire(key, REDIS_EXPIRE)
	return nil
}
