module github.com/theforeman/ygg_worker

go 1.16

require (
	git.sr.ht/~spc/go-log v0.1.1
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/redhatinsights/yggdrasil v0.4.2
	github.com/redhatinsights/yggdrasil_v0 v0.0.0-20210811162724-41397343c25b
	google.golang.org/grpc v1.59.0
)

replace github.com/redhatinsights/yggdrasil_v0 v0.0.0-20210811162724-41397343c25b => ./yggdrasil-41397343c25b8664a1da79ccd74a02f562c03f24
