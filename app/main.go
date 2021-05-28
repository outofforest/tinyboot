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

	ctx, exit := tinyboot.Configure()
	defer exit()

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

	select {
	case <-time.After(10 * time.Second):
		panic("reboot test")
	case <-ctx.Done():
		fmt.Println("Context canceled")
		<-time.After(5 * time.Second)
	}

}
