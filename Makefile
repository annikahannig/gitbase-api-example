
PROG=gitbase-api

.PHONY: clean $(PROG)

all: deps $(PROG)

deps:
	go get .

test:
	go test -v

$(PROG):
	go build -o $(PROG)


clean:
	rm -f $(PROG)

