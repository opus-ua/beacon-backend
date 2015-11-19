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

func (db *DBClient) AddBeacon(post *Beacon, userID uint64) (uint64, error) {
	id, err := db.AddBeaconRedis(post, userID)
	// post.AddPostGres()
	return id, err
}

func (db *DBClient) AddComment(comment *Comment, userID uint64) error {
	db.AddCommentRedis(comment, userID)
	// comment.AddPostGres()
	return nil
}

func (db *DBClient) HeartPost(postID uint64, userID uint64) error {
	return db.HeartPostRedis(postID, userID)
}

func (db *DBClient) FlagPost(postID uint64, userID uint64) error {
	return db.FlagPostRedis(postID, userID)
}

func (db *DBClient) CreateUser(username string, authkey []byte, email string) (uint64, error) {
	return db.CreateUserRedis(username, authkey, email)
}

func (db *DBClient) UserExists(userid uint64) (bool, error) {
	return db.UserExistsRedis(userid)
}

func (db *DBClient) UserAuthenticated(userid uint64, authkey []byte) (bool, error) {
	if exists, _ := db.UserExists(userid); !exists {
		return false, nil
	}
	if db.devMode {
		return true, nil
	}
	return db.UserAuthenticatedRedis(userid, authkey)
}

func (db *DBClient) GetUsername(userid uint64) (string, error) {
	return db.GetUsernameRedis(userid)
}

func (db *DBClient) UsernameExists(username string) (bool, error) {
	return db.UsernameExistsRedis(username)
}

func (db *DBClient) EmailExists(email string) (bool, error) {
	return db.EmailExistsRedis(email)
}

func (db *DBClient) GetUserIDByEmail(email string) (uint64, error) {
	return db.GetUserIDByEmailRedis(email)
}

func (db *DBClient) SetUserAuthKey(userid uint64, authkey []byte) error {
	return db.SetUserAuthKeyRedis(userid, authkey)
}
