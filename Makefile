build:
	docker build -t shreddedbacon/concoursebot .
	if [ ! -d builds ]; then mkdir builds; fi
	docker run --name concoursebot --rm -v `pwd`:/project shreddedbacon/concoursebot

run:
	if [ -f ./builds/concoursebot ]; then	./builds/concoursebot; fi
