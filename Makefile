HOME := $(HOME)

SYSTEM := $(shell uname -s)
ifeq ($(SYSTEM), Linux)
	HOME := /home/ubuntu
endif

all: start

pre:
	@mkdir -p $(HOME)/data/caddy
	@mkdir -p $(HOME)/data/backend

start: pre
	@sudo docker compose -f ./docker-compose.yml up -d

stop:
	@sudo docker compose -f ./docker-compose.yml down

clean:
	@sudo docker compose -f ./docker-compose.yml down -v --rmi all

fclean: clean
	@sudo rm -rf $(HOME)/data/caddy/*
	@sudo rm -rf $(HOME)/data/backend/*

.PHONY: all pre start stop clean fclean