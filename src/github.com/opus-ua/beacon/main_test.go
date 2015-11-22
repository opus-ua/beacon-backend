package main

import (
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	go StartServer(true, true)
	time.Sleep(50 * time.Millisecond)
	res := m.Run()
	os.Exit(res)
}

func TestGetVersion(t *testing.T) {
	resp, err := http.Get("http://localhost:8765/version")
	if err != nil {
		t.Fatalf("Could not connect to beacon backend.")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Could not parse response body.")
	}
	if !strings.Contains(string(body), "version") ||
		strings.Contains(string(body), "error") {
		t.Fatalf("Response body did not contain 'version'.")
	}

	t.Log(string(body))
}

var imgData = `
ffd8ffe000104a46494600010101004800480000fffe0013437265617465
6420776974682047494d50ffdb0043000302020302020303030304030304
050805050404050a070706080c0a0c0c0b0a0b0b0d0e12100d0e110e0b0b
1016101113141515150c0f171816141812141514ffdb0043010304040504
0509050509140d0b0d141414141414141414141414141414141414141414
1414141414141414141414141414141414141414141414141414141414ff
c20011080001000103011100021101031101ffc400140001000000000000
00000000000000000008ffc4001401010000000000000000000000000000
0000ffda000c03010002100310000001549fffc400141001000000000000
00000000000000000000ffda00080101000105027fffc400141101000000
00000000000000000000000000ffda0008010301013f017fffc400141101
00000000000000000000000000000000ffda0008010201013f017fffc400
14100100000000000000000000000000000000ffda0008010100063f027f
ffc40014100100000000000000000000000000000000ffda000801010001
3f217fffda000c030100020003000000109fffc400141101000000000000
00000000000000000000ffda0008010301013f107fffc400141101000000
00000000000000000000000000ffda0008010201013f107fffc400141001
00000000000000000000000000000000ffda0008010100013f107fffd9
`

var jsonData = `
    {
        "userid": 1,
        "text": "*high pitched squealing*",
        "longitude": 45.0,
        "latitude": 45.0
    }
`

func TestPostBeacon(t *testing.T) {
	imgData = strings.Replace(imgData, "\n", "", -1)
	imgBytes, err := hex.DecodeString(imgData)
	if err != nil {
		t.Fatalf("Unable to parse image data.")
	}
	body := &bytes.Buffer{}
	partWriter := multipart.NewWriter(body)
	jsonHeader := textproto.MIMEHeader{}
	jsonHeader.Add("Content-Type", "application/json")
	jsonWriter, err := partWriter.CreatePart(jsonHeader)
	io.WriteString(jsonWriter, jsonData)
	imgHeader := textproto.MIMEHeader{}
	imgHeader.Add("Content-Type", "img/jpeg")
	imgWriter, err := partWriter.CreatePart(imgHeader)
	imgWriter.Write(imgBytes)
	partWriter.Close()
	req, _ := http.NewRequest("POST", "http://localhost:8765/beacon", body)
	req.Header.Add("Content-Type", partWriter.FormDataContentType())
	req.SetBasicAuth("1", "0")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(err.Error())
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err.Error())
	}
	if !strings.Contains(string(respBody), "*high pitched squealing*") {
		t.Fatalf("Response did not contain correct content: \n%s", string(respBody))
	}
}

func TestGetBeacon(t *testing.T) {
	resp, err := http.Get("http://localhost:8765/beacon/1")
	if err != nil {
		t.Fatalf("Could not connect to beacon backend.")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Response status code was %d.", resp.StatusCode)
	}
}

func TestHeartPost(t *testing.T) {
	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8765/heart/1", &bytes.Buffer{})
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth("1", "0")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Could not connect to beacon backend.")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Response status code was %d.", resp.StatusCode)
	}
}

func TestFlagPost(t *testing.T) {
	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8765/flag/1", &bytes.Buffer{})
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth("1", "0")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Could not connect to beacon backend.")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Response status code was %d.", resp.StatusCode)
	}
}
