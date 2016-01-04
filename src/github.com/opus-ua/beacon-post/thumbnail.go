package beaconpost

import (
	"bytes"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
)

const maxWidth uint = 200
const maxHeight uint = 300

func MakeThumbnail(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return []byte{}, err
	}
	thumb := resize.Thumbnail(maxWidth, maxHeight, img, resize.Lanczos3)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, thumb, nil)
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}
