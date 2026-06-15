module github.com/zapmarket/zapmarket/pkg/database

go 1.22

require (
	github.com/lib/pq v1.12.3
	github.com/zapmarket/zapmarket/pkg/config v0.0.0
)

replace github.com/zapmarket/zapmarket/pkg/config => ../config
