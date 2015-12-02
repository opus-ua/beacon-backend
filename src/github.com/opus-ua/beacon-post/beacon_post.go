package beaconpost

import (
	"time"
)

type Beacon struct {
	ID          uint64
	Image       []byte
	Thumbnail   []byte
	Location    Geotag
	PosterID    uint64
	Description string
	Hearts      uint32
	Flags       uint32
	Time        time.Time
	Comments    []Comment
}

type Comment struct {
	ID       uint64
	PosterID uint64
	BeaconID uint64
	Text     string
	Hearts   uint32
	Flags    uint32
	Time     time.Time
}
