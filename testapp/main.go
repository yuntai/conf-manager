package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/websocket"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func echoHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func AsyncRun() {
}

func main() {
	msgCh := make(chan string)
	onConnected := func(ws *websocket.Conn) {
		defer func() {
			ws.Close()
		}()
		ws.Write([]byte("connected"))

		fmt.Printf("waiting...\n")
		for {
			select {
			case val := <-msgCh:
				fmt.Printf("received...\n")
				ws.Write([]byte(val))
			}
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	// initial
	dat, err := ioutil.ReadFile("testapp.json")
	check(err)
	go func() {
		defer func() {
			fmt.Printf("out of func\n")
		}()
		select {
		case msgCh <- string(dat):
			fmt.Printf("sent\n")
		}
	}()

	go func() {
		for sig := range c {
			println(sig)
			fmt.Printf("signal(%s) reloading conf....\n", sig)
			dat, err := ioutil.ReadFile("testapp.json")
			check(err)
			fmt.Print(string(dat))
			msgCh <- string(dat)
		}
	}()

	http.Handle("/echo", websocket.Handler(onConnected))
	http.Handle("/", http.FileServer(http.Dir(".")))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
	fmt.Printf("port(%d)\n", 8080)
}
