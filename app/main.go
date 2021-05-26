package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wojciech-malota-wojcik/tinyboot"
)

func main() {
	// ignore logs produced by dhclient
	log.SetOutput(ioutil.Discard)

	tinyboot.Configure()

	for {
		func() {
			resp, err := http.Get("https://www.google.com")
			if err != nil {
				fmt.Println(err)
			} else {
				defer resp.Body.Close()
				_, err = io.Copy(os.Stdout, resp.Body)
				if err != nil {
					panic(err)
				}
			}
		}()
		<-time.After(time.Second)
	}
}
