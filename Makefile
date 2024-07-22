APP_ENV := DEV
HOST_PORT := 3000

build:
	sudo docker build -t app .
run:
	sudo -E docker run -e APP_ENV=${APP_ENV} -p ${HOST_PORT}:3000 --rm -ti app
.PHONY:
	build run