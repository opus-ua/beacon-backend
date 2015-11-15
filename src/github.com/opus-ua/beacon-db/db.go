package beacondb

import (
	. "github.com/opus-ua/beacon-post"
	"gopkg.in/redis.v3"
)

type DBClient struct {
	redis *redis.Client
	// postgres *postgres.Client
	devMode bool
	err     error
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
		redis:   DefaultRedisDB(),
		devMode: false,
	}
}

func DevDB() *DBClient {
	db := &DBClient{
		redis:   DevRedisDB(),
		devMode: true,
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

func (db *DBClient) CreateUser(username string, authkey []byte) (uint64, error) {
	return db.CreateUserRedis(username, authkey)
}

func (db *DBClient) UserExists(userid uint64) bool {
	return db.UserExistsRedis(userid)
}

func (db *DBClient) UserAuthenticated(userid uint64, authkey []byte) bool {
	if !db.UserExists(userid) {
		return false
	}
	if db.devMode {
		return true
	}
	return db.UserAuthenticatedRedis(userid, authkey)
}

func (db *DBClient) GetUsername(userid uint64) (string, error) {
	return db.GetUsernameRedis(userid)
}
