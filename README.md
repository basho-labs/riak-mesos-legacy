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