package main

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"regexp"
	"strings"
	"testing"
)

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
	AppendStringToFile(testFileName, l + "\r\n")

	fh, _ := os.Open(testFileName)
	defer fh.Close()
	fScan := bufio.NewScanner(fh)
	for fScan.Scan() {
		lf := fScan.Text()
		//fmt.Println(lf)
		assert.Equal(t,l,lf,"same line")
	}

}
