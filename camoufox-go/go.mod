module github.com/uzzalhcse/camoufox-go

go 1.22

require (
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/oschwald/geoip2-golang v1.13.0
	github.com/playwright-community/playwright-go v0.5200.1
	github.com/uzzalhcse/browserforge-go v0.0.0
)

require (
	github.com/deckarep/golang-set/v2 v2.7.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/oschwald/maxminddb-golang v1.13.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
)

replace github.com/uzzalhcse/browserforge-go => ../browserforge-go
