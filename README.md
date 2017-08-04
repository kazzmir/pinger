Ncurses UI for sending multiple pings at the same time

![pinger](https://github.com/kazzmir/pinger/raw/master/pinger-small.png)

Build
```
$ make setup
$ make
$ ./pinger
```

Example usage
```
$ pinger google.com cnn.com facebook.com
$ pinger 192.168.0.0/24
```

Go libraries used:

go get github.com/sparrc/go-ping

go get github.com/nsf/termbox-go

This library attempts to send an "unprivileged" ping via UDP. On linux, this must be enabled by setting

```
sudo sysctl -w net.ipv4.ping_group_range="0   2147483647"
```

If you do not wish to do this, you can set pinger.SetPrivileged(true) and use setcap to allow your binary using go-ping to bind to raw sockets (or just run as super-user):

```
setcap cap_net_raw=+ep /bin/goping-binary
```
