namedyn
=======

namedyn is a simple dynamic dns client for name.com (unofficial), written in golang. It only supports IPv4.

# build
```bash
# you need to have golang installed to build your binary
go build && mv namedyn /usr/local/bin/
```

# usage
```bash
# please note: you need to login to name.com and create an api token fist
# ---
# to handle home.example.com
USERNAME=username TOKEN=xxxxxxxxx DOMAIN=example.com HOST=home namedyn
```
