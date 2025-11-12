module github.com/appetiteclub/appetite/services/table

go 1.24.7

require (
	github.com/aquamarinepk/aqm v0.0.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/google/uuid v1.6.0
	go.mongodb.org/mongo-driver v1.17.6
)

replace (
	github.com/appetiteclub/appetite/pkg/lib/auth => ../../pkg/lib/auth
	github.com/appetiteclub/appetite/pkg/lib/core => ../../pkg/lib/core
)
