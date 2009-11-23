include $(GOROOT)/src/Make.$(GOARCH)

TARG=client
GOFILES=\
	client.go\

include $(GOROOT)/src/Make.cmd
