// Copyright © 2010 Fazlul Shahriar <fshahriar@gmail.com>.
// See LICENSE file for license details.

package main

import "fmt"
import "os"
import "plan9/p"
import "plan9/p/clnt"

func fatal(format string, v ...) {
	fmt.Fprintf(os.Stderr, format, v)
	os.Exit(1)
}

func MountP9P(name string) (c *clnt.Clnt, e *p.Error) {
	uname, err := os.Getenverror("LOGNAME")
	if err != nil {
		fatal("$LOGNAME not set\n")
	}
	user := p.OsUsers.Uname2User(uname)
	ns, err := Getns()
	if err != nil {
		fatal("could not get name space: %s\n", err)
	}
	return clnt.Mount("unix", ns+"/"+name, "", user)
}

func main() {
	acme, err := MountP9P("acme")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to acme: %s\n", err)
		os.Exit(1)
	}
	file, err := acme.FOpen("new", 0)
	if err != nil {
		fatal("open new: %s\n", err)
	}
	file.Close()
	acme.Unmount()
}
