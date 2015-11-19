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
	REDIS_EXPIRE = 86400 * time.Second
	// REDIS_EXPIRE   = -1
	REDIS_INT_BASE    = 10
	USERNAME_POOL_KEY = "usernames"
	USER_COUNT_KEY    = "user-count"
)

func DefaultRedisDB() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
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

func GetRedisUserKey(id uint64) string {
	return fmt.Sprintf("u:%d", id)
}

func GetRedisUserEmailKey(email string) string {
	return fmt.Sprintf("email:%s", email)
}

func GetRedisUserHeartedKey(postid uint64) string {
    return fmt.Sprintf("h:%d", postid)
}

func GetRedisUserFlaggedKey(postid uint64) string {
    return fmt.Sprintf("f:%d", postid)
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

func (db *DBClient) AddBeaconRedis(post *Beacon, userID uint64) (uint64, error) {
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

func (db *DBClient) AddCommentRedis(comment *Comment, userID uint64) error {
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

func (db *DBClient) HeartPostRedis(postID uint64, userID uint64) error {
    poolKey := GetRedisUserHeartedKey(postID)
    setMem := fmt.Sprintf("%d", userID)
    res, err := db.redis.SIsMember(poolKey, setMem).Result()
    if res || err != nil {
        return errors.New("Post has already been hearted.")
    }
    _, err = db.redis.SAdd(poolKey, setMem).Result()
    if err != nil {
        return err
    }
	key := GetRedisPostKey(postID)
	_, err = db.redis.HIncrBy(key, "hearts", 1).Result()
	if err != nil {
		return err
	}
	db.redis.Expire(key, REDIS_EXPIRE)
	return nil
}

func (db *DBClient) FlagPostRedis(postID uint64, userID uint64) error {
    poolKey := GetRedisUserFlaggedKey(postID)
    setMem := fmt.Sprintf("%d", userID)
    res, err := db.redis.SIsMember(poolKey, setMem).Result()
    if res || err != nil {
        return errors.New("Post has already been flagged.")
    }
    _, err = db.redis.SAdd(poolKey, setMem).Result()
    if err != nil {
        return err
    }
	key := GetRedisPostKey(postID)
	_, err = db.redis.HIncrBy(key, "flags", 1).Result()
	if err != nil {
		return err
	}
	db.redis.Expire(key, REDIS_EXPIRE)
	return nil
}

func (db *DBClient) CreateUserRedis(username string, authkey []byte, email string) (uint64, error) {
	if res, err := db.redis.SIsMember(USERNAME_POOL_KEY, username).Result(); res || err != nil {
		return 0, errors.New("Username already exists.")
	}
	return db.AddUserRedis(username, authkey, email)
}

func (db *DBClient) AddUserRedis(username string, authkey []byte, email string) (uint64, error) {
	userIDSigned, err := db.redis.Incr(USER_COUNT_KEY).Result()
	userID := uint64(userIDSigned)
	if err != nil {
		return 0, errors.New("Could not get number of users in db.")
	}
	if db.SetUserRedis(userID, username, authkey, email) != nil {
		return 0, errors.New("Could not add user to db.")
	}
	if db.redis.SAdd(USERNAME_POOL_KEY, username).Err() != nil {
		return 0, errors.New("Could not reserve username.")
	}
	if db.redis.Set(GetRedisUserEmailKey(email), userID, 0).Err() != nil {
		return 0, errors.New("Could not add email to pool.")
	}
	return userID, nil
}

func (db *DBClient) SetUserRedis(userID uint64, username string, authkey []byte, email string) error {
	userKey := GetRedisUserKey(userID)
	now := time.Now().Format(time.UnixDate)
	res := db.redis.HMSet(userKey, "id", strconv.FormatUint(userID, REDIS_INT_BASE),
		"username", username,
		"created", now,
		"flags-rec", "0",
		"flags-sub", "0",
		"hearts-rec", "0",
		"hearts-sub", "0",
		"auth", string(authkey),
		"email", email)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (db *DBClient) UserExistsRedis(userid uint64) (bool, error) {
	res, err := db.redis.Exists(GetRedisUserKey(userid)).Result()
	if err != nil {
		return false, err
	}
	return res, nil
}

func (db *DBClient) UsernameExistsRedis(username string) (bool, error) {
	res, err := db.redis.SIsMember(USERNAME_POOL_KEY, username).Result()
	if err != nil {
		return false, err
	}
	return res, nil
}

func (db *DBClient) EmailExistsRedis(email string) (bool, error) {
	res, err := db.redis.Exists(GetRedisUserEmailKey(email)).Result()
	if err != nil {
		return false, err
	}
	return res, nil
}

func (db *DBClient) GetUserIDByEmailRedis(email string) (uint64, error) {
	res, err := db.redis.Get(GetRedisUserEmailKey(email)).Result()
	if err != nil {
		return 0, err
	}
	id, err := RedisParseUInt64(res, nil)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (db *DBClient) UserAuthenticatedRedis(userid uint64, authkey []byte) (bool, error) {
	storedKey, err := db.redis.HGet(GetRedisUserKey(userid), "auth").Result()
	if err != nil {
		return false, err
	}
	return storedKey == string(authkey), nil
}

func (db *DBClient) GetUsernameRedis(userid uint64) (string, error) {
	return db.redis.HGet(GetRedisUserKey(userid), "username").Result()
}

func (db *DBClient) SetUserAuthKeyRedis(userid uint64, authkey []byte) error {
	return db.redis.HSet(GetRedisUserKey(userid), "auth", string(authkey)).Err()
}
