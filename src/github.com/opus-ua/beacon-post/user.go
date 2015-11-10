package beaconpost

import (
	"time"
)

type User struct {
	ID              uint64
	Username        string
	AccountCreated  time.Time
	FlagsReceived   uint32
	HeartsReceived  uint32
	FlagsSubmitted  uint32
	HeartsSubmitted uint32
}

type UserProfile struct {
	User
	Beacons []Beacon
}
