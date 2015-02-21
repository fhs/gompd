// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file

// Package server implements a fake MPD server that's used to test gompd client.
package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"os"
	"strconv"
	"strings"
)

// Attrs is a set of attributes returned by MPD.
type Attrs map[string]string

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

type playlist struct {
	songs []int
}

func newPlaylist() *playlist {
	return &playlist{songs: make([]int, 0)}
}

func (p *playlist) At(i int) int {
	return p.songs[i]
}

func (p *playlist) Len() int {
	return len(p.songs)
}

func (p *playlist) Add(song int) {
	p.songs = append(p.songs, song)
}

func (p *playlist) Delete(i int) {
	if i < 0 || i >= len(p.songs) {
		return
	}
	copy(p.songs[i:], p.songs[i+1:])
	p.songs = p.songs[:len(p.songs)-1]
}

func (p *playlist) Clear() {
	p.songs = p.songs[:0]
}

func (p *playlist) Append(q *playlist) {
	// TODO: do at most one allocation
	for i := 0; i < q.Len(); i++ {
		p.Add(q.At(i))
	}
}

type server struct {
	state           string
	database        []Attrs        // database of songs
	index           map[string]int // maps URI to database index
	playlists       map[string]*playlist
	currentPlaylist *playlist
	pos             int // in currentPlaylist
	idleEventc      chan string
	idleStartc      chan *idleRequest
	idleEndc        chan uint
}

func newServer() *server {
	s := &server{
		state:           "stop",
		database:        make([]Attrs, 100),
		index:           make(map[string]int, 100),
		playlists:       make(map[string]*playlist),
		currentPlaylist: newPlaylist(),
		pos:             0,
		idleEventc:      make(chan string),
		idleStartc:      make(chan *idleRequest),
		idleEndc:        make(chan uint),
	}
	for i := 0; i < len(s.database); i++ {
		s.database[i] = make(Attrs, 5)
		filename := fmt.Sprintf("song%04d.ogg", i)
		s.database[i]["file"] = filename
		s.index[filename] = i
	}
	return s
}

func (s *server) writeResponse(p *textproto.Conn, args []string, okLine string) (cmdOk, closed bool) {
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
	case "lsinfo":
		for _, a := range s.database {
			p.PrintfLine("file: %s", a["file"])
			p.PrintfLine("Last-Modified: 2014-07-02T12:32:26Z")
			p.PrintfLine("Artist: Newcleus")
			p.PrintfLine("Title: Jam On It")
			p.PrintfLine("Track: 02")
		}
		for _, a := range []string{
			"music/Buck 65 - Dirtbike 1",
			"music/Howlin' Wolf - Moanin' in the Moonlight",
		} {
			p.PrintfLine("directory: %s", a)
		}
		p.PrintfLine("playlist: BBC 6 Music.m3u")
	case "listplaylists":
		for k := range s.playlists {
			p.PrintfLine("playlist: %s", k)
		}
	case "playlistinfo":
		var rng []string
		var start int
		end := s.currentPlaylist.Len()

		if len(args) >= 2 {
			rng = strings.Split(args[1], ":")
		}

		if len(rng) == 1 {
			// Requesting a single song from the playlist at position i.
			i, err := strconv.Atoi(rng[0])
			if err != nil {
				ack("invalid song position")
				return
			}
			start = i
			end = i + 1
		} else if len(rng) == 2 {
			// Requesting a range of the playlist from specified start/end positions.
			var err error
			start, err = strconv.Atoi(rng[0])
			if err != nil {
				ack("Integer or range expected")
				return
			}
			end, err := strconv.Atoi(rng[1])
			if err != nil {
				ack("Integer or range expected")
				return
			}
			if start < 0 || end < 0 {
				ack("Number is negative")
				return
			}
		}

		for i := start; i < end; i++ {
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
			s.playlists[name] = newPlaylist()
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
		if pos < 0 || pos >= pl.Len() {
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
	case "delete":
		if len(args) != 2 {
			ack("wrong number of arguments")
			return
		}
		i, err := strconv.Atoi(args[1])
		if err != nil {
			ack("invalid song position")
			return
		}
		s.idleEventc <- "playlist"
		if i < 0 || i >= s.currentPlaylist.Len() {
			ack("invalid song position")
			return
		}
		s.currentPlaylist.Delete(i)
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
		s.playlists[name] = newPlaylist()
		s.playlists[name].Append(s.currentPlaylist)
	case "play", "stop":
		s.idleEventc <- "player"
		s.state = args[0]
	case "next":
		s.idleEventc <- "player"
		if s.pos < 0 || s.pos >= s.currentPlaylist.Len() {
			s.pos = 0
			break
		}
		s.pos = (s.pos + 1) % s.currentPlaylist.Len()
	case "previous":
		s.idleEventc <- "player"
		if s.pos < 0 || s.pos >= s.currentPlaylist.Len() {
			s.pos = 0
			break
		}
		if s.pos == 0 {
			s.pos = s.currentPlaylist.Len() - 1
			break
		}
		s.pos--
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
	case "outputs":
		p.PrintfLine("outputid: 0")
		p.PrintfLine("outputenabled: 1")
		p.PrintfLine("outputname: downstairs")
		p.PrintfLine("outputid: 1")
		p.PrintfLine("outputenabled: 0")
		p.PrintfLine("outputname: upstairs")
	case "disableoutput", "enableoutput":
	default:
		p.PrintfLine("ACK {} unknown command %q", args[0])
		log.Printf("unknown command: %s\n", args[0])
		return
	}
	cmdOk = true
	p.PrintfLine(okLine)
	return
}

type requestType int

const (
	simple requestType = iota
	commandListOk
	idle
	noIdle
)

type request struct {
	typ     requestType
	args    []string
	cmdList [][]string
}

func (s *server) readRequest(p *textproto.Conn) (*request, error) {
	line, err := p.ReadLine()
	if err == io.EOF {
		return nil, err
	}
	if err != nil {
		log.Printf("reading request failed: %v\n", err)
		return nil, err
	}
	args := parseArgs(line)
	if len(args) == 0 {
		return &request{typ: simple, args: args}, nil
	}
	switch args[0] {
	case "command_list_ok_begin":
		var cmdList [][]string
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
		return &request{typ: commandListOk, cmdList: cmdList}, nil

	case "idle":
		return &request{typ: idle, args: args}, nil

	case "noidle":
		return &request{typ: noIdle, args: args}, nil
	}
	return &request{typ: simple, args: args}, nil
}

type idleRequest struct {
	endTokenc  chan uint   // for token used to end event broadcast
	eventc     chan string // for subsystem name
	subsystems []string    // subsystems to listen for changes
}

func (s *server) writeIdleResponse(p *textproto.Conn, id uint, quit chan bool, subsystems []string) {
	p.StartResponse(id)
	defer p.EndResponse(id)

	req := &idleRequest{
		endTokenc:  make(chan uint),
		eventc:     make(chan string, 1),
		subsystems: subsystems,
	}
	s.idleStartc <- req
	token := <-req.endTokenc
	select {
	case name := <-req.eventc:
		p.PrintfLine("changed: %s", name)
		p.PrintfLine("OK")
		<-quit
	case <-quit:
		p.PrintfLine("OK")
	}
	s.idleEndc <- token
}

func (s *server) handleConnection(p *textproto.Conn) {
	id := p.Next()
	p.StartRequest(id)
	p.EndRequest(id)
	p.StartResponse(id)
	p.PrintfLine("OK MPD gompd0.1")
	p.EndResponse(id)

	endIdle := make(chan bool)
	inIdle := false
	defer p.Close()
	for {
		id := p.Next()
		p.StartRequest(id)
		req, err := s.readRequest(p)
		if err != nil {
			return
		}
		// We need to do this inside request because idle response
		// may not have ended yet, but it will end after the following.
		if inIdle {
			endIdle <- true
		}
		p.EndRequest(id)

		if req.typ == idle {
			inIdle = true
			go s.writeIdleResponse(p, id, endIdle, req.args[1:])
			// writeIdleResponse does it's own StartResponse/EndResponse
			continue
		}

		p.StartResponse(id)
		if inIdle {
			inIdle = false
		}
		switch req.typ {
		case noIdle:
		case commandListOk:
			var ok, closed bool
			ok = true
			for _, args := range req.cmdList {
				ok, closed = s.writeResponse(p, args, "list_OK")
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
		case simple:
			if _, closed := s.writeResponse(p, req.args, "OK"); closed {
				return
			}
		}
		p.EndResponse(id)
	}
}

var knownSubsystems = []string{
	"database",
	"update",
	"stored_playlist",
	"playlist",
	"player",
	"mixer",
	"output",
	"options",
}

func indexID(v []uint, id uint) int {
	for i, n := range v {
		if id == n {
			return i
		}
	}
	return -1
}

func deleteID(v []uint, id uint) []uint {
	i := indexID(v, id)
	if i < 0 {
		return v
	}
	copy(v[i:], v[i+1:])
	return v[:len(v)-1]
}

func (s *server) broadcastIdleEvents() {
	clientChans := make(map[uint]chan string)
	subsys := make(map[string][]uint)
	for _, name := range knownSubsystems {
		subsys[name] = make([]uint, 0)
	}
	token := uint(0)
	for {
		select {
		case req := <-s.idleStartc:
			clientChans[token] = req.eventc
			names := req.subsystems
			if len(req.subsystems) == 0 {
				names = knownSubsystems
			}
			for _, name := range names {
				if _, ok := subsys[name]; !ok {
					subsys[name] = make([]uint, 0)
				}
				subsys[name] = append(subsys[name], token)
			}
			req.endTokenc <- token
			token++

		case client := <-s.idleEndc:
			delete(clientChans, client)
			for name := range subsys {
				subsys[name] = deleteID(subsys[name], client)
			}

		case name := <-s.idleEventc:
			if clients, ok := subsys[name]; ok {
				for _, c := range clients {
					select {
					case clientChans[c] <- name:
					default:
					}
				}
			}
		}
	}
}

// Listen starts the server on the network network and address addr.
// Once the server has started, a value is sent to listening channel.
func Listen(network, addr string, listening chan bool) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		log.Fatalf("Listen failed: %v\n", err)
		os.Exit(1)
	}
	s := newServer()
	go s.broadcastIdleEvents()
	listening <- true
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept failed: %v\n", err)
			continue
		}
		go s.handleConnection(textproto.NewConn(conn))
	}
}
