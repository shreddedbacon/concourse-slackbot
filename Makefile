build:
	docker build -t shreddedbacon/concoursebot:build -f Dockerfile.build .
	if [ ! -d builds ]; then mkdir builds; fi
	docker run --name concoursebot --rm -v `pwd`:/project shreddedbacon/concoursebot:build

build-run:
	docker build -t shreddedbacon/concoursebot:latest -f Dockerfile .
	if [ -f ./config.json ]; then	docker run --name concoursebot --rm -v `pwd`/config.json:/app/config.json shreddedbacon/concoursebot:latest; fi

run:
	if [ -f ./builds/concoursebot ]; then	./builds/concoursebot; fi
