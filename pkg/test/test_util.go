package test

import (
	"fmt"
	"net"
	"net/http"

	"github.com/google/uuid"
)

type UnixSocketServer struct {
	addr string
	mux  *http.ServeMux
	l    net.Listener
}

func NewUnixSocketServer() *UnixSocketServer {
	return &UnixSocketServer{
		addr: fmt.Sprintf("@%s", uuid.New().String()),
		mux:  http.NewServeMux(),
	}
}

func (uss *UnixSocketServer) GetAddress() string {
	return fmt.Sprintf("unix://%s", uss.addr)
}

func (uss *UnixSocketServer) RegisterHandler(path string, handler http.HandlerFunc) {
	uss.mux.HandleFunc(path, handler)
}

func (uss *UnixSocketServer) Start() error {
	l, err := net.Listen("unix", uss.addr)
	if err != nil {
		return err
	}

	uss.l = l

	return http.Serve(l, uss.mux)
}

func (uss *UnixSocketServer) Stop() error {
	if uss.l != nil {
		return uss.l.Close()
	}
	return nil
}
