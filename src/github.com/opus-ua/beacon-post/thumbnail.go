package beaconpost

import (
	"bytes"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
)

const maxWidth uint = 100
const maxHeight uint = 150

func MakeThumbnail(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return []byte{}, err
	}
	thumb := resize.Resize(maxWidth, maxHeight, img, resize.Lanczos3)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, thumb, nil)
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}
