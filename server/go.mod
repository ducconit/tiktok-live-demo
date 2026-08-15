module github.com/tiktok-bar/server

go 1.26.4

require (
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	github.com/steampoweredtaco/gotiktoklive v0.0.4
)

replace github.com/steampoweredtaco/gotiktoklive => ./third_party/gotiktoklive

require (
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/erni27/imcache v1.2.1 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.uber.org/ratelimit v0.3.1 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
