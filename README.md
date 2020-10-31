Simple script which fetches [immuni-app](https://github.com/immuni-app) exposure data from backend and calculates the total number of TEKs reported.
Protobuf interface definition has been borrowed from [github.com/google/exposure-notifications-server](https://github.com/google/exposure-notifications-server).

Usage:

```
$ go get github.com/marcoguerri/immuni-stats
$ go run $GOPATH/src/github.com/marcoguerri/immuni-stats/main.go
```

