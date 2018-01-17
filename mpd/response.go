// Copyright 2018 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package mpd

import "fmt"

// A Command represents a MPD command.
type Command struct {
	client *Client
	cmd    string
}

// TODO: automatically quote strings

// Command returns a command that can be sent to MPD sever.
// It enables low-level access to MPD protocol and should be avoided if
// the user is not familiar with MPD protocol.
func (c *Client) Command(format string, args ...interface{}) *Command {
	return &Command{
		client: c,
		cmd:    fmt.Sprintf(format, args...),
	}
}

// OK sends command to server and checks for error.
func (cmd *Command) OK() error {
	id, err := cmd.client.cmd(cmd.cmd)
	if err != nil {
		return err
	}
	cmd.client.text.StartResponse(id)
	defer cmd.client.text.EndResponse(id)
	return cmd.client.readOKLine("OK")
}

// Attrs sends command to server and reads attributes returned in response.
func (cmd *Command) Attrs() (Attrs, error) {
	id, err := cmd.client.cmd(cmd.cmd)
	if err != nil {
		return nil, err
	}
	cmd.client.text.StartResponse(id)
	defer cmd.client.text.EndResponse(id)
	return cmd.client.readAttrs("OK")
}

// AttrsList sends command to server and reads a list of attributes returned in response.
// Each attribute group starts with key startKey.
func (cmd *Command) AttrsList(startKey string) ([]Attrs, error) {
	id, err := cmd.client.cmd(cmd.cmd)
	if err != nil {
		return nil, err
	}
	cmd.client.text.StartResponse(id)
	defer cmd.client.text.EndResponse(id)
	return cmd.client.readAttrsList(startKey)
}

// Strings sends command to server and reads a list of strings returned in response.
// Each string have the key key.
func (cmd *Command) Strings(key string) ([]string, error) {
	id, err := cmd.client.cmd(cmd.cmd)
	if err != nil {
		return nil, err
	}
	cmd.client.text.StartResponse(id)
	defer cmd.client.text.EndResponse(id)
	return cmd.client.readList(key)
}
