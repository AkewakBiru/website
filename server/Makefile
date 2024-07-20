build:
	sudo docker build -t reverse_proxy .
run:
	sudo -E docker run -e UPSTREAMS=${UPSTREAMS} -p 80:80 -p 443:443 -p 443:443/udp -p 2019:2019 --rm -ti reverse_proxy
.PHONY:
	build run