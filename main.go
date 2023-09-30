package main

import (
  "net/http"
  "net/http/httputil"
  "net/url"
  "fmt"
  "os"
)

type Server interface {
  Address() string  // returns the address with which to access the server

  // returns true if the server is alive and able to serve request
  IsAlive() bool

  // uses this server to process the request
  Serve(w http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
  addr string
  proxy *httputil.ReverseProxy
}

func (s *simpleServer) Address() string { return s.addr }

func (s *simpleServer) IsAlive() bool { return true }

func (s *simpleServer) Serve(w http.ResponseWriter, r *http.Request) {
  s.proxy.ServeHTTP(w, r)
}

func newSimpleServer(addr string) *simpleServer {
  u, err := url.Parse(addr)
  handleErr(err)
  return &simpleServer{
    addr: addr,
    proxy: httputil.NewSingleHostReverseProxy(u),
  }
}

type loadbalancer struct {
  port string
  rrCount int // round robin count
  servers []Server
}

func NewLoadbalancer(port string, servers []Server) *loadbalancer {
  return &loadbalancer{
    port: port,
    rrCount: 0,
    servers: servers,
  }
}

func (lb *loadbalancer) getNextAvailableServer() Server {
  server := lb.servers[lb.rrCount % len(lb.servers)]
  for !server.IsAlive() {
    lb.rrCount++
    server = lb.servers[lb.rrCount % len(lb.servers)]
  }
  lb.rrCount++

  return server
}

func (lb *loadbalancer) serveProxy(w http.ResponseWriter, r *http.Request) {
  targetServer := lb.getNextAvailableServer()
  fmt.Printf("forwarding request to address %q\n", targetServer.Address())
  targetServer.Serve(w, r)
}

func main() {
  servers := []Server{
    newSimpleServer("https://www.facebook.com"),
    newSimpleServer("https://www.bing.com"),
    newSimpleServer("https://www.google.com"),
  }
  lb := NewLoadbalancer("8000", servers)
  handleRedirect := func(w http.ResponseWriter, r *http.Request) {
    lb.serveProxy(w, r)
  }
  http.HandleFunc("/", handleRedirect)
  fmt.Printf("serving requests at localhost:%s\n", lb.port)
  http.ListenAndServe(":"+lb.port, nil)
}

func handleErr(err error) {
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}
