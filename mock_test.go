package httpclient

import _ "github.com/golang/mock/mockgen/model"

//go:generate mockgen --build_flags=--mod=mod -destination=./mock/http.go -package=mock -mock_names=Handler=Handler net/http Handler
