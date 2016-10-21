package main

import (
	"fmt"
	"log"
	"net/http"

	githttp "github.com/AaronO/go-git-http"
)

// GitHTTPServer is internal git http server serving local git clients
type GitHTTPServer struct {
	port       int
	url        string
	path       string
	shutdownCh chan struct{}
}

// Run starts git http handler
func (g *GitHTTPServer) Run() error {
	handler := githttp.New(g.path)
	http.Handle("/", handler)

	//TODO: handling error from ListenAndServe
	go func() {
		addr := fmt.Sprintf(":%d", g.port)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Printf("Failed to start http(%v)\n", err)
			log.Fatal("fatal")
		} else {
			fmt.Printf("start http(%v)\n", err)
		}
	}()
	return nil
}

// NewGitHTTPServer creates new http server
func NewGitHTTPServer(localPath string, port int) *GitHTTPServer {
	return &GitHTTPServer{
		port:       port,
		url:        fmt.Sprintf("http://localhost:%d", port),
		path:       localPath,
		shutdownCh: make(chan struct{}),
	}
}
