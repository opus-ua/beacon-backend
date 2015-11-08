package beaconpost

import (
	"bytes"
	"encoding/binary"
	"math"
)

type Geotag struct {
	Latitude  float64
	Longitude float64
}

func (tag *Geotag) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, tag.Latitude)
	if err != nil {
		return []byte{}, err
	}
	err = binary.Write(buf, binary.LittleEndian, tag.Longitude)
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func (tag *Geotag) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &tag.Latitude)
	if err != nil {
		return err
	}
	err = binary.Read(buf, binary.LittleEndian, &tag.Longitude)
	if err != nil {
		return err
	}
	return nil
}

func ToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// Calculates great circle distance between
// two geotags in *kilometers*
func Distance(p1 Geotag, p2 Geotag) float64 {
	delta := Geotag{
		Latitude:  ToRadians(p2.Latitude - p1.Latitude),
		Longitude: ToRadians(p2.Longitude - p1.Longitude),
	}
	a := math.Pow(math.Sin(delta.Latitude/2.0), 2) +
		math.Cos(ToRadians(p1.Latitude)) +
		math.Cos(ToRadians(p2.Latitude)) +
		math.Pow(math.Sin(delta.Longitude/2.0), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))
	return c * 6371.0
}

func KilometersToMiles(km float64) float64 {
	return 0.621371 * km
}
