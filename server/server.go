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
	"sync"
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
	mu              sync.RWMutex
}

func NewServer() *Server {
	s := &Server{
		state:           "stop",
		database:        make([]Attrs, 100),
		index:           make(map[string]int, 100),
		playlists:       make(map[string]*Playlist),
		currentPlaylist: NewPlaylist(),
	}
	for i := 0; i < len(s.database); i++ {
		s.database[i] = make(Attrs, 5)
		filename := fmt.Sprintf("song%04d.ogg", i)
		s.database[i]["file"] = filename
		s.index[filename] = i
	}
	return s
}

func (s *Server) writeResponse(p *textproto.Conn, id uint, args []string) (closed bool) {
	p.StartResponse(id)
	defer p.EndResponse(id)

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
		s.mu.Lock()
		s.state = args[0]
		s.mu.Lock()
	case "pause":
		s.mu.Lock()
		if s.state != "stop" {
			s.state = args[0]
		}
		s.mu.Lock()
	case "status":
		s.mu.RLock()
		state := s.state
		s.mu.RLock()
		p.PrintfLine("state: %s", state)
	case "update":
		p.PrintfLine("updating_db: 1")
	case "ping":
	default:
		log.Printf("unknown command: %s\n", args[0])
	}
	p.PrintfLine("OK")
	return
}

func (s *Server) handleConnection(p *textproto.Conn) {
	id := p.Next()
	p.StartRequest(id)
	p.EndRequest(id)
	p.StartResponse(id)
	p.PrintfLine("OK MPD gompd0.1")
	p.EndResponse(id)

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
