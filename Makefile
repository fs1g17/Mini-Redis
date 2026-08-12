run-redis:
	docker run --rm -it redis redis-cli -h host.docker.internal -p 6379 $(CMD)