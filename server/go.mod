module github.com/tiktok-bar/server

go 1.26.4

require (
	github.com/bogdanfinn/fhttp v0.6.8
	github.com/bogdanfinn/tls-client v1.15.1
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	github.com/refraction-networking/utls v1.8.2
	github.com/steampoweredtaco/gotiktoklive v0.0.4
)

replace github.com/steampoweredtaco/gotiktoklive => ./third_party/gotiktoklive

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/bdandy/go-errors v1.2.2 // indirect
	github.com/bdandy/go-socks4 v1.2.3 // indirect
	github.com/bogdanfinn/quic-go-utls v1.0.9-utls // indirect
	github.com/bogdanfinn/utls v1.7.7-barnius // indirect
	github.com/bogdanfinn/websocket v1.5.5-barnius // indirect
	github.com/erni27/imcache v1.2.1 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.1.0 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/tam7t/hpkp v0.0.0-20160821193359-2b70b4024ed5 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
