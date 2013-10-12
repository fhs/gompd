// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"os"
	"strconv"
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

type Attrs map[string]string

type Playlist struct {
	songs []int
}

func NewPlaylist() *Playlist {
	return &Playlist{songs: make([]int, 0)}
}

func (p *Playlist) At(i int) int {
	return p.songs[i]
}

func (p *Playlist) Len() int {
	return len(p.songs)
}

func (p *Playlist) Add(song int) {
	p.songs = append(p.songs, song)
}

func (p *Playlist) Delete(i int) {
	copy(p.songs[i:], p.songs[i+1:])
	p.songs = p.songs[:len(p.songs)-1]
}

func (p *Playlist) Clear() {
	p.songs = p.songs[:0]
}

func (p *Playlist) Append(q *Playlist) {
	// TODO: do at most one allocation
	for i := 0; i < q.Len(); i++ {
		p.Add(q.At(i))
	}
}

type Server struct {
	state           string
	database        []Attrs        // database of songs
	index           map[string]int // maps URI to database index
	playlists       map[string]*Playlist
	currentPlaylist *Playlist
	pos             int // in currentPlaylist
}

func NewServer() *Server {
	s := &Server{
		state:           "stop",
		database:        make([]Attrs, 100),
		index:           make(map[string]int, 100),
		playlists:       make(map[string]*Playlist),
		currentPlaylist: NewPlaylist(),
		pos:             0,
	}
	for i := 0; i < len(s.database); i++ {
		s.database[i] = make(Attrs, 5)
		filename := fmt.Sprintf("song%04d.ogg", i)
		s.database[i]["file"] = filename
		s.index[filename] = i
	}
	return s
}

func (s *Server) writeResponse(p *textproto.Conn, id uint, args []string, okLine string) (cmdOk, closed bool) {
	if len(args) < 1 {
		p.PrintfLine("No command given")
		return
	}
	ack := func(format string, a ...interface{}) error {
		return p.PrintfLine("ACK {"+args[0]+"} "+format, a...)
	}
	switch args[0] {
	case "close":
		closed = true
		return
	case "list":
		if len(args) < 2 {
			ack("too few arguments")
			return
		}
		if args[1] == "file" {
			for _, a := range s.database {
				p.PrintfLine("file: %s", a["file"])
			}
		}
	case "listplaylists":
		for k := range s.playlists {
			p.PrintfLine("playlist: %s", k)
		}
	case "playlistinfo":
		if len(args) >= 2 {
			i, err := strconv.Atoi(args[1])
			if err != nil {
				ack("invalid song position")
				return
			}
			p.PrintfLine("file: %s", s.database[s.currentPlaylist.At(i)]["file"])
			break
		}
		for i := 0; i < s.currentPlaylist.Len(); i++ {
			p.PrintfLine("file: %s", s.database[s.currentPlaylist.At(i)]["file"])
		}
	case "listplaylistinfo":
		if len(args) < 2 {
			ack("too few arguments")
			return
		}
		pl, ok := s.playlists[args[1]]
		if !ok {
			ack("no such playlist")
			return
		}
		for i := 0; i < pl.Len(); i++ {
			p.PrintfLine("file: %s", s.database[pl.At(i)]["file"])
		}
	case "playlistadd":
		if len(args) != 3 {
			ack("wrong number of arguments")
			return
		}
		name, uri := args[1], args[2]
		i, ok := s.index[uri]
		if !ok {
			ack("URI not found")
			return
		}
		if s.playlists[name] == nil {
			s.playlists[name] = NewPlaylist()
		}
		s.playlists[name].Add(i)
	case "playlistdelete":
		if len(args) != 3 {
			ack("wrong number of arguments")
			return
		}
		name := args[1]
		pos, err := strconv.Atoi(args[2])
		if err != nil {
			ack("invalid position number")
			return
		}
		pl, ok := s.playlists[name]
		if !ok {
			ack("playlist not found")
			return
		}
		if pos >= pl.Len() {
			ack("invalid song position")
			return
		}
		pl.Delete(pos)
	case "playlistclear":
		if len(args) != 2 {
			ack("wrong number of arguments")
			return
		}
		pl, ok := s.playlists[args[1]]
		if !ok {
			ack("playlist not found")
			return
		}
		pl.Clear()
	case "rm":
		if len(args) != 2 {
			ack("wrong number of arguments")
			return
		}
		_, ok := s.playlists[args[1]]
		if !ok {
			ack("playlist not found")
			return
		}
		delete(s.playlists, args[1])
	case "rename":
		if len(args) != 3 {
			ack("wrong number of arguments")
			return
		}
		old, new := args[1], args[2]
		_, ok := s.playlists[old]
		if !ok {
			ack("playlist %s does not exist", old)
			return
		}
		_, ok = s.playlists[new]
		if ok {
			ack("playlist %s already exists", new)
			return
		}
		s.playlists[new] = s.playlists[old]
		delete(s.playlists, old)
	case "load":
		if len(args) != 2 {
			ack("wrong number of arguments")
			return
		}
		pl, ok := s.playlists[args[1]]
		if !ok {
			ack("playlist %s does not exist", args[1])
			return
		}
		s.currentPlaylist.Append(pl)
	case "clear":
		s.currentPlaylist.Clear()
	case "add":
		if len(args) != 2 {
			ack("wrong number of arguments")
			return
		}
		i, ok := s.index[args[1]]
		if !ok {
			ack("URI not found")
			return
		}
		s.currentPlaylist.Add(i)
	case "save":
		if len(args) != 2 {
			ack("wrong number of arguments")
			return
		}
		name := args[1]
		_, ok := s.playlists[name]
		if ok {
			ack("playlist %s already exists", name)
			return
		}
		s.playlists[name] = NewPlaylist()
		s.playlists[name].Append(s.currentPlaylist)
	case "play", "stop":
		s.state = args[0]
	case "next":
		if s.pos < 0 || s.pos >= s.currentPlaylist.Len() {
			s.pos = 0
			break
		}
		s.pos = (s.pos + 1) % s.currentPlaylist.Len()
	case "previous":
		if s.pos < 0 || s.pos >= s.currentPlaylist.Len() {
			s.pos = 0
			break
		}
		if s.pos == 0 {
			s.pos = s.currentPlaylist.Len() - 1
			break
		}
		s.pos -= 1
	case "pause":
		if s.state != "stop" {
			s.state = args[0]
		}
	case "status":
		state := s.state
		p.PrintfLine("state: %s", state)
	case "update":
		p.PrintfLine("updating_db: 1")
	case "ping":
	case "currentsong":
		if s.currentPlaylist.Len() == 0 {
			break
		}
		if s.pos >= s.currentPlaylist.Len() {
			s.pos = 0
		}
		p.PrintfLine("file: %s", s.database[s.currentPlaylist.At(s.pos)]["file"])
	default:
		p.PrintfLine("ACK {} unknown command %q", args[0])
		log.Printf("unknown command: %s\n", args[0])
		return
	}
	cmdOk = true
	p.PrintfLine(okLine)
	return
}

type RequestType int

const (
	CommandListOk RequestType = iota
	Simple
)

type Request struct {
	typ     RequestType
	args    []string
	cmdList [][]string
}

func (s *Server) readRequest(p *textproto.Conn) (*Request, error) {
	line, err := p.ReadLine()
	if err == io.EOF {
		return nil, err
	}
	if err != nil {
		log.Printf("reading request failed: %v\n", err)
		return nil, err
	}
	args := parseArgs(line)
	if len(args) > 0 && args[0] == "command_list_ok_begin" {
		cmdList := make([][]string, 0)
		for {
			line, err := p.ReadLine()
			if err == io.EOF {
				return nil, err
			}
			if err != nil {
				log.Printf("reading request failed: %v\n", err)
				return nil, err
			}
			args = parseArgs(line)
			if len(args) > 0 && args[0] == "command_list_end" {
				break
			}
			cmdList = append(cmdList, args)
		}
		return &Request{typ: CommandListOk, cmdList: cmdList}, nil
	}
	return &Request{typ: Simple, args: args}, nil
}

func (s *Server) handleConnection(p *textproto.Conn) {
	id := p.Next()
	p.StartRequest(id)
	p.EndRequest(id)
	p.StartResponse(id)
	p.PrintfLine("OK MPD gompd0.1")
	p.EndResponse(id)

	defer p.Close()
	for {
		id := p.Next()
		p.StartRequest(id)
		req, err := s.readRequest(p)
		if err != nil {
			return
		}
		p.EndRequest(id)

		p.StartResponse(id)
		switch req.typ {
		case CommandListOk:
			var ok, closed bool
			ok = true
			for _, args := range req.cmdList {
				ok, closed = s.writeResponse(p, id, args, "list_OK")
				if closed {
					return
				}
				if !ok {
					break
				}
			}
			if ok {
				p.PrintfLine("OK")
			}
		case Simple:
			if _, closed := s.writeResponse(p, id, req.args, "OK"); closed {
				return
			}
		}
		p.EndResponse(id)
	}
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
