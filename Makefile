# Copyright © 2009 Fazlul Shahriar <fshahriar@gmail.com>.
# See LICENSE file for license details.

include $(GOROOT)/src/Make.$(GOARCH)

TARG=mpd
GOFILES=\
	client.go\

include $(GOROOT)/src/Make.pkg
