# Shell to use with Make
SHELL := /bin/bash

# Export targets not associated with files.
.PHONY: protobuf

# Compile protocol buffers
protobuf:
	@echo "compiling protocol buffers"
	@protoc -I ping/ ping/*.proto --go_out=plugins=grpc:ping/
