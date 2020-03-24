run_example:
	docker build -t ytuploader ./ -f example/Dockerfile
	docker run ytuploader