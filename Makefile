include $(GOROOT)/src/Make.$(GOARCH)

TARG=mpd
GOFILES=\
	client.go\

include $(GOROOT)/src/Make.pkg
