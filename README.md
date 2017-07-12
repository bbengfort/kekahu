# KeKahu

**Keep alive client for the Kahu service.**

KeKahu is a client service for the [Kahu API](https://github.com/bbengfort/kahu) that manages hosts on our experimental test bed. KeKahu's primary role is to send a keep-alive heartbeat to Kahu at a routine interval. KeKahu also provides [DDNS](https://en.wikipedia.org/wiki/Dynamic_DNS) by using an public IP lookup and posting that IP address to Kahu. Finally KeKahu can be used to synchronize peers and access the Kahu API in a meaningful way.

## Getting Started

As long as you have [go version 1.8](https://golang.org/dl/) or later installed you can get and install KeKahu as follows:

```
$ go get github.com/bbengfort/kekahu/...
```

This command will build and install the kekahu binary in `$GOBIN`. In order for the command to be used with [systemd](https://wiki.ubuntu.com/SystemdForUpstartUsers), however, you must install it into a system path such as `/usr/local/bin`. I recommend using a symlink to make sure that the latest binary is used as follows:

```
$ sudo ln -s $GOBIN/kekahu /usr/local/bin/kekahu
```

Now that KeKahu is installed you can see its commands and options:

```
$ kekahu --help
```

Most of KeKahu is configured through the environment, though command line options can be specified. A `.env` file can be used for local use or development. Ensure the following environmental variables are set:

- `$KEKAHU_API_KEY`: the Kahu API key for this machine
- `$KEKAHU_URL` (optional): url of the Kahu API
- `$KEKAHU_INTERVAL` (optional): interval between heartbeats
- `$PEERS_PATH` (optional): location on disk to store network peers.

Once these environment variables are set, you can use the kekahu application. For example, to synchronize network peers:

```
$ kekahu sync
```

## Systemd

Kekahu is configured to be managed by systemd on Linux systems. To get started create a file in `/etc/systemd/system/kekahu.service` as follows:

```
[Unit]
Description=KeKahu Service
Documentation=https://github.com/bbengfort/kekahu

[Service]
Type=simple
Environment=KEKAHU_API_KEY=mykey
Environment=KEKAHU_URL=myurl
Environment=KEKAHU_INTERVAL=myinterval
Environment=PEERS_PATH=mypath
ExecStart=/usr/local/bin/kekahu start
ExecReload=/usr/local/bin/kekahu reload
ExecStop=/usr/local/bin/kekahu stop
Restart=on-abort

[Install]
WantedBy=multi-user.target
```

Now reload the services and enable the kekahu service:

```
$ sudo systemctl enable kekahu
```

The service can be managed with the `start`, `stop`, `reload`, and `status` commands as follows:

```
$ sudo systemctl daemon-reload
$ sudo systemctl start kekahu
```

You can check the status of the service to see if it started correctly, or use kekahu directly to check the status. In order to view the log files use the following command:

```
$ sudy journalctl -u kekahu
```

This should show everything written to stdout and stderr from the application.
