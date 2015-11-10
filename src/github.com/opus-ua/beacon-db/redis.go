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

type DBClient struct {
	redis *redis.Client
	// postgres *postgres.Client
	err error
}

func NewDB(dev bool) *DBClient {
	if dev {
		return DevDB()
	} else {
		return DefaultDB()
	}
}

func DefaultDB() *DBClient {
	return &DBClient{
		redis: DefaultRedisDB(),
	}
}

func DefaultRedisDB() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

func DevDB() *DBClient {
	db := &DBClient{
		redis: DevRedisDB(),
	}
	AddDummy(db)
	return db
}

func DevRedisDB() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       11,
	})
	client.FlushDb()
	return client
}

func GetRedisPostKey(id uint64) string {
	return fmt.Sprintf("p:%d", id)
}

func GetRedisCommentListKey(id uint64) string {
	return fmt.Sprintf("%s:c", GetRedisPostKey(id))
}

func (db *DBClient) GetThread(id uint64) (Beacon, error) {
	return db.GetThreadRedis(id)
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

func (db *DBClient) GetBeaconRedis(id uint64) (Beacon, error) {
	key := GetRedisPostKey(id)
	res, err := db.redis.HGetAllMap(key).Result()
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

func (db *DBClient) GetCommentRedis(id uint64, parent uint64) (Comment, error) {
	commentKey := GetRedisPostKey(id)
	commHash, err := db.redis.HGetAllMap(commentKey).Result()
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

func (db *DBClient) GetCommentListRedis(id uint64) ([]uint64, error) {
	key := GetRedisCommentListKey(id)
	strList, err := db.redis.LRange(key, 0, -1).Result()
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

func (db *DBClient) GetThreadRedis(id uint64) (Beacon, error) {
	post, err := db.GetBeaconRedis(id)
	if err != nil {
		return Beacon{}, err
	}
	comments, err := db.GetCommentListRedis(id)
	if err != nil {
		return Beacon{}, err
	}
	for _, commentID := range comments {
		comment, err := db.GetCommentRedis(commentID, id)
		if err != nil {
			return Beacon{}, err
		}
		post.Comments = append(post.Comments, comment)
	}
	return post, nil
}

func (db *DBClient) AddBeacon(post *Beacon) (uint64, error) {
	id, err := db.AddBeaconRedis(post)
	// post.AddPostGres()
	return id, err
}

func (db *DBClient) AddBeaconRedis(post *Beacon) (uint64, error) {
	postID, err := db.redis.Incr("post-count").Result()
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
	db.redis.HMSet(key, "img", string(post.Image[:]),
		"loc", locString,
		"poster", strconv.FormatUint(post.PosterID, REDIS_INT_BASE),
		"desc", post.Description,
		"hearts", strconv.FormatUint(uint64(post.Hearts), REDIS_INT_BASE),
		"flags", strconv.FormatUint(uint64(post.Flags), REDIS_INT_BASE),
		"time", now,
		"type", "beacon")
	db.redis.Expire(key, REDIS_EXPIRE)
	return post.ID, nil
}

func (db *DBClient) AddComment(comment *Comment) error {
	db.AddCommentRedis(comment)
	// comment.AddPostGres()
	return nil
}

func (db *DBClient) AddCommentRedis(comment *Comment) error {
	commentID, err := db.redis.Incr("post-count").Result()
	if commentID < 0 {
		return errors.New("Retrieved post count was negative.")
	}
	comment.ID = uint64(commentID)
	if err != nil {
		return err
	}
	beaconKey := GetRedisPostKey(comment.BeaconID)
	IDKey := GetRedisCommentListKey(comment.BeaconID)
	db.redis.RPush(IDKey, strconv.FormatUint(comment.ID, REDIS_INT_BASE))
	commKey := GetRedisPostKey(comment.ID)
	now := time.Now().Format(time.UnixDate)
	db.redis.HMSet(commKey, "poster", strconv.FormatUint(comment.PosterID, REDIS_INT_BASE),
		"parent", strconv.FormatUint(comment.BeaconID, REDIS_INT_BASE),
		"text", comment.Text,
		"hearts", strconv.FormatUint(uint64(comment.Hearts), REDIS_INT_BASE),
		"flags", strconv.FormatUint(uint64(comment.Flags), REDIS_INT_BASE),
		"time", now,
		"type", "comment")
	db.redis.Expire(beaconKey, REDIS_EXPIRE)
	db.redis.Expire(IDKey, REDIS_EXPIRE)
	db.redis.Expire(commKey, REDIS_EXPIRE)
	return nil
}

func (db *DBClient) HeartPost(postID uint64) error {
	return db.HeartPostRedis(postID)
}

func (db *DBClient) HeartPostRedis(postID uint64) error {
	key := GetRedisPostKey(postID)
	_, err := db.redis.HIncrBy(key, "hearts", 1).Result()
	if err != nil {
		return err
	}
	db.redis.Expire(key, REDIS_EXPIRE)
	return nil
}

func (db *DBClient) FlagPost(postID uint64) error {
	return db.FlagPostRedis(postID)
}

func (db *DBClient) FlagPostRedis(postID uint64) error {
	key := GetRedisPostKey(postID)
	_, err := db.redis.HIncrBy(key, "flags", 1).Result()
	if err != nil {
		return err
	}
	db.redis.Expire(key, REDIS_EXPIRE)
	return nil
}
