package beacondb

import (
	"gopkg.in/redis.v3"
	. "github.com/opus-ua/beacon-post"
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

func DevDB() *DBClient {
	db := &DBClient{
		redis: DevRedisDB(),
	}
	AddDummy(db)
	return db
}

func (db *DBClient) GetThread(id uint64) (Beacon, error) {
	return db.GetThreadRedis(id)
	/*
	   if post, err = GetPostGres(id, gresClient); err == nil {
	       return post, nil
	   }
	*/
}

func (db *DBClient) AddBeacon(post *Beacon) (uint64, error) {
	id, err := db.AddBeaconRedis(post)
	// post.AddPostGres()
	return id, err
}

func (db *DBClient) AddComment(comment *Comment) error {
	db.AddCommentRedis(comment)
	// comment.AddPostGres()
	return nil
}

func (db *DBClient) HeartPost(postID uint64) error {
	return db.HeartPostRedis(postID)
}

func (db *DBClient) FlagPost(postID uint64) error {
	return db.FlagPostRedis(postID)
}
