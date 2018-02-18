package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func init() {
	rand.Seed(1) // fix rand fo tests
}

func deleteFile(file string) {
	// delete file
	var err = os.Remove(file)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
}

func TestLog(t *testing.T) {
	testFileName := "_test.log"
	defer deleteFile(testFileName)

	o := []byte(`{"Text":"some *text*\r\n","EV":"eve","Source":"some |source| \n "}`)

	day, l := ParseEntry(o)
	//fmt.Println(d + " " + l)

	elem := strings.Split(l, "|")
	assert.Equal(t, day, elem[0], "test day")
	assert.Equal(t, "eve", elem[2], "eve")
	assert.Equal(t, "some *text*", elem[3], "some *text*")
	assert.Equal(t, "Origine some source  ", elem[4], "Origine some source  ")

	// Test FormatHTMLLine(line string) (string, string)
	var re = regexp.MustCompile(`<span class="t">(.*?)</span>`)
	d2, lh := FormatHTMLLine(l)
	//fmt.Println(d2 + " " + lh)

	res := re.FindStringSubmatch(lh)
	assert.Equal(t, d2, day, "test day")
	assert.Equal(t, "some <b>text</b>", res[1], "some <b>text</b>")

	// Test AppendStringToFile(path, text string) error
	AppendStringToFile(testFileName, l+"\r\n")

	fh, _ := os.Open(testFileName)
	defer fh.Close()
	fScan := bufio.NewScanner(fh)
	for fScan.Scan() {
		lf := fScan.Text()
		//fmt.Println(lf)
		assert.Equal(t, l, lf, "same line")
	}

}

func contains(s []string, t string) bool {
	for _, v := range s {
		if v == t {
			return true
		}
	}
	return false
}

func TestAsset(t *testing.T) {
	defer func() {
		deleteFile("t/web/index.html")
		deleteFile("t/web/")
		deleteFile("t/")
	}()
	_, err := Asset("web/index.html")
	if err != nil {
		// asset was not found.
		fmt.Println(err)
	}
	d, _ := AssetInfo("web/index.html")
	assert.Equal(t, "web/index.html", d.Name(), "name of asset")
	a := AssetNames()
	liste := []string{
		"web/index.html",
	}
	for _, la := range a {
		assert.Equal(t, true, contains(liste, la), "asset in list")
	}

	a, _ = AssetDir("web/med")
	liste = []string{"default.min.css", "medium-editor.min.css", "medium-editor.min.js"}
	for _, la := range a {
		assert.Equal(t, true, contains(liste, la), "asset in list")
	}
	RestoreAssets("t/", "web/index.html")
	_, e := os.Stat("t/web/index.html")
	assert.Equal(t, nil, e, "restored file exist")
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestServer(t *testing.T) {
	testFileName := "_test.log"
	defer deleteFile(testFileName)

	localPort := "5000"
	randomPass := RandStringBytes(12)
	assert.Equal(t, "ynvpoqhs5g3x", randomPass, "test 12 char rand string")

	gin.SetMode(gin.TestMode)
	r := gin.New()

	assert.Equal(t, "http://exemple.com", localServer("http://exemple.com", localPort), "don't change fixed server url")
	urlContains := regexp.MustCompile(`:5000/share/$`).MatchString

	localServUrl := localServer("", localPort)
	assert.Equal(t, true, urlContains(localServUrl), "create local server url")

	banner(localPort, localServUrl, randomPass, version)

	server(r, "http://exemple.com", "crise", "qwerty", testFileName, true)

	/**
	  test template
	  **/
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		fmt.Println(err)
	}

	resp1 := httptest.NewRecorder()
	r.ServeHTTP(resp1, req)
	//fmt.Printf("%+v\n", resp1.Body)
	assert.Equal(t, 200, resp1.Code, "template success")

	/**
	  test basic auth
	  **/
	req, err = http.NewRequest("GET", "/share", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth("crise", "qwerty"))
	if err != nil {
		fmt.Println(err)
	}

	resp1 = httptest.NewRecorder()
	r.ServeHTTP(resp1, req)
	//fmt.Printf("%+v\n", resp1.Body)
	assert.Equal(t, 200, resp1.Code, "basic auth success")

	/**
	  test websocket
	  **/

	s := httptest.NewServer(r)
	defer s.Close()

	d := websocket.Dialer{}
	c, resp, err := d.Dial("ws://"+s.Listener.Addr().String()+"/log/ws", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode, "ok switching connect")

	/**
	test share info
	**/
	o1 := []byte(`share`)
	err = c.WriteMessage(websocket.TextMessage, o1)
	if err != nil {
		t.Fatal(err)
	}

	_, sharews, _ := c.ReadMessage()
	fmt.Printf("%+v\n", string(sharews))
	assert.Equal(t, "share--http://exemple.com--qwerty", string(sharews), "load contains some <b>text</b>")

	/**
	init file with some text
	**/
	o := []byte(`{"Text":"some *text*\r\n","EV":"eve","Source":"some |source| \n "}`)
	//err = c.WriteJSON(o)
	err = c.WriteMessage(websocket.TextMessage, o)
	if err != nil {
		t.Fatal(err)
	}

	_, respws, _ := c.ReadMessage()
	//c.ReadJSON(&respws)
	//fmt.Printf("%+v\n", string(respws))

	n := time.Now()
	day := n.Format("02-01-2006")
	dt := "<span class=\"day\">" + day + ":</span>"
	assert.Equal(t, dt, string(respws), "test return passwd")

	/**
	test load full file
	**/
	textContains := regexp.MustCompile(`some <b>text</b>`).MatchString

	o = []byte(`load`)
	err = c.WriteMessage(websocket.TextMessage, o)
	if err != nil {
		t.Fatal(err)
	}

	_, loadws, _ := c.ReadMessage()
	//fmt.Printf("%+v\n", string(loadws))
	sload := string(loadws)
	assert.Equal(t, true, textContains(sload), "load contains some <b>text</b>")

}
