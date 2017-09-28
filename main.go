package main

/*

./go-bindata -o myweb.go web/index.html


go build  -ldflags "-s" -o crisislog *.go

*/

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"
)

var version string = "0.0.1"

func AppendStringToFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(text)
	if err != nil {
		return err
	}
	return nil
}

func FormatHTMLLine(line string) string {
	var re = regexp.MustCompile(`^(.*?)\|(.*?)\|(.*?)\|(.*?)\|(.*?)$`)
	var reB = regexp.MustCompile(`<span class="t">(.*?)\*(.*?)\*(.*?)</span>`)
	// XXX XSS, TODO parser, filter, escape
	nlt := re.ReplaceAllString(line, `$1 <span class="$2">$2</span> <span class="t">$3</span> <span class="src">$4</span> <span class="auth">$5</span>`)
	nl := reB.ReplaceAllString(nlt, `$1<b>$2</b>$3`)
	return nl
}

//const letterBytes = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
const letterBytes = "abcdefghijkmnopqrstuvwxyz23456789" // simpliest password

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	pass := RandStringBytes(8)

	servPtr := flag.String("s", "", "Serveur")
	usrPtr := flag.String("u", "crise", "Utilisateur")
	filePtr := flag.String("f", "./chrono.log", "Fichier de log")
	portPtr := flag.String("p", "5000", "Port")
	debugPtr := flag.Bool("d", false, "Debug mode")
	flag.Parse()

	p := *portPtr
	user := *usrPtr
	file := *filePtr
	serv := *servPtr
	debug := *debugPtr

	if debug == false {
		gin.SetMode(gin.ReleaseMode)
	}

	// Config server
	r := gin.New()

	r.Use(gin.Recovery())
	if debug == true {
		r.Use(gin.Logger())
	}
	m := melody.New()
	m.Config.MaxMessageSize = 65536 //2^16

	addrs, _ := net.InterfaceAddrs()

	if serv == "" {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				serv = ipnet.IP.String()
				if ipnet.IP.To4() != nil { // prefer shorter IPv4 if available
					break
				}
			}
		}
		serv = "http://" + serv + ":" + p + "/share/"
	}

	fmt.Println("#--------------------------------------------#")
	fmt.Println(" ")
	fmt.Println("    Usage =>  http://localhost:" + p + "/  <=")
	fmt.Println(" ")
	fmt.Println("  Partage :")
	fmt.Println("  =========")
	fmt.Println("  Server: " + serv)
	fmt.Println("    Pass: " + pass)
	fmt.Println(" ")
	fmt.Println("  version: " + version)
	fmt.Println("#--------------------------------------------#")

	// Add Asset
	data, err := Asset("web/index.html")
	if err != nil {
		// asset was not found.
		fmt.Println(err)
	}

	// Manage share auth
	auth := r.Group("/", gin.BasicAuthForRealm(gin.Accounts{
		user: pass,
	}, "Utilisateur: "+user))

	// Gin router
	auth.GET("/share", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	r.GET("/", func(c *gin.Context) {
		if c.ClientIP() == "::1" {
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		}
	})

	// Websocket router
	r.GET("/ws", func(c *gin.Context) {
		ml := make(map[string]interface{})
		ml["cip"] = c.ClientIP()
		m.HandleRequestWithKeys(c.Writer, c.Request, ml)
	})

	// Manage websocket messages
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		l, _ := s.Get("cip")
		if string(msg) == "share" {
			// display share access
			var as []*melody.Session
			as = append(as, s)
			byteArray := []byte("share--" + serv + "--" + pass)
			m.BroadcastMultiple(byteArray, as)
		} else if string(msg) == "load" {
			// read full log
			var as []*melody.Session
			as = append(as, s)
			fh, _ := os.Open(file)
			defer fh.Close()
			fScan := bufio.NewScanner(fh)
			for fScan.Scan() {
				byteArray := []byte(FormatHTMLLine(fScan.Text()))
				m.BroadcastMultiple(byteArray, as)
			}
		} else {
			var objmap map[string]*json.RawMessage
			e := json.Unmarshal(msg, &objmap)
			if e != nil {
				fmt.Println(e)
			}
			te := *objmap["Text"]
			ev := *objmap["EV"]
			sc := *objmap["Source"]
			//fmt.Println("sc ",len(sc))
			//fmt.Println("ev ",string(ev))
			src := ""
			if len(sc) > 2 {
				src = fmt.Sprintf("Origine %s", sc[1:len(sc)-1])
			}
			t := time.Now()
			line := fmt.Sprintf("%s|%s|%s|%s|%s",
				t.Format("15:04:05"),
				ev[1:len(ev)-1], //remove quotes
				te[1:len(te)-1], //remove quotes
				src,
				l)
			// append to file
			err := AppendStringToFile(file, line+"\r\n")
			if err == nil {
				byteArray := []byte(FormatHTMLLine(line))
				m.Broadcast(byteArray)
			} else {
				fmt.Println(err)
			}
		}
	})

	r.Run(":" + p)
}
