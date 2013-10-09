// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file

package main

import (
	"io"
	"log"
	"net"
	"net/textproto"
	"os"
)

func unquote(line string, start int) (string, int) {
	i := start
	if line[i] != '"' {
		for i < len(line) && (line[i] != ' ' && line[i] != '\t') {
			i++
		}
		return line[start:i], i
	}

	i++ // beginning "
	s := make([]byte, len(line[i:]))
	n := 0
	for i < len(line) {
		if line[i] == '"' { // ending "
			i++
			break
		}
		if line[i] == '\\' && i+1 < len(line) {
			i++
		}
		s[n] = line[i]
		i++
		n++
	}
	return string(s[:n]), i
}

func parseArgs(line string) (args []string) {
	var s string
	i := 0
	for i < len(line) {
		if line[i] == ' ' || line[i] == '\t' {
			i++
			continue
		}
		s, i = unquote(line, i)
		args = append(args, s)
	}
	return
}

type Server struct {
	state string
}

func NewServer() *Server {
	return &Server{state: "stop"}
}

func (s *Server) writeResponse(p *textproto.Conn, id uint, args []string) (closed bool) {
	p.StartResponse(id)
	defer p.EndResponse(id)

	if len(args) < 1 {
		p.PrintfLine("No command given")
		return
	}
	switch args[0] {
	case "close":
		closed = true
	case "status":
		p.PrintfLine("state: %s", s.state)
		p.PrintfLine("OK")
	case "play", "stop":
		s.state = args[0]
		p.PrintfLine("OK")
	case "pause":
		if s.state == "stop" {
			p.PrintfLine("OK")
			return
		}
		s.state = args[0]
		p.PrintfLine("OK")
	default:
		//for i, arg := range args {
		//	p.PrintfLine("arg %d is %q", i, arg)
		//}
		p.PrintfLine("OK")
	}
	return
}

func (s *Server) handleConnection(p *textproto.Conn) {
	for {
		id := p.Next()
		p.StartRequest(id)
		line, err := p.ReadLine()
		p.EndRequest(id)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("reading request failed: %v\n", err)
			break
		}
		args := parseArgs(line)
		if s.writeResponse(p, id, args) {
			break
		}
	}
	p.Close()
}

func main() {
	ln, err := net.Listen("tcp", ":6600")
	if err != nil {
		log.Fatalf("Listen failed: %v\n", err)
		os.Exit(1)
	}
	s := NewServer()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept failed: %v\n", err)
			continue
		}
		go s.handleConnection(textproto.NewConn(conn))
	}
}
