package beaconpost

import (
    "time"
)

type BeaconPost struct {
    ID          uint64
    Image       []byte
    Location    Geotag
    PosterID    uint64
    Description string
    Hearts      uint32
    Flags       uint32
    Time        time.Time
    Comments    []Comment
}

type BeaconThumb struct {
    ID          uint64
    Thumb       []byte
    Location    Geotag
    PosterID    uint64
    Description string
    Hearts      uint32
    Time        time.Time
}

type Comment struct {
    ID          uint64
    PosterID    uint64
    BeaconID    uint64
    Text        string
    Hearts      uint32
    Flags       uint32
    Time        time.Time
}
