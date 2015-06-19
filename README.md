# Build instructions

This is the beginning of the Mesos Go Riak framework. Go was chosen because it's
an easy to use systems language and it has much better semantics around safety as
compared to Java.

```
1. Set your GOPATH to the location you checked out Bletchley
2. Make a src/github.com/mesos directory, and cd into it
3. git clone https://github.com/mesos/mesos-go.git
4. cd mesos-go
5. godep restore
6. go build ./...
7. Check out Bletchley into WORKSPACE/src/github.com/basho/

```

Building Bletchley itself:
```
1. cd into WORKSPACE/src/github.com/basho/bletchley/bin
# Make sure you have gox installed, from here: https://github.com/mitchellh/gox
2. gox -osarch="linux/amd64" -osarch=darwin/amd64 ../...
```

TODO: Eventually, we should use go-bindata and go generate to embed the executor in the scheduler. But for now, I'm lazy.
(Reference: https://github.com/jteeuwen/go-bindata)
Running Bletchley: `./scheduler_darwin_amd64` on Mac OS X or `./scheduler_linux_amd64` on Linux


Directory structure at the end:
```
bletchley/src/github.com/basho/bletchley/.git
bletchley/src/github.com/mesos/mesos-go/.git
```