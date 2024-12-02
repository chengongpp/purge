CC=go build
CFLAGS=

all: gitdump svndump dsstoredump redisuck

clean:
	rm -rf target

gitdump:
	mkdir -p target
	cd gitdump
	$(CC) -o ../target/gitdump gitdump.go

svndump:
	mkdir -p target
	cd svndump
	$(CC) -o ../target/svndump svndump.go

dsstoredump:
	mkdir -p target
	cd dsstoredump
	$(CC) -o ../target/dsstoredump dsstoredump.go

redisuck:
	mkdir -p target
	$(CC) -o target/redisuck redisuck/redisuck.go