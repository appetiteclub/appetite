module github.com/appetiteclub/appetite/cmd/utils

go 1.23.0

require (
	github.com/appetiteclub/appetite/services/kitchen v0.0.0
	github.com/appetiteclub/appetite/services/order v0.0.0
	github.com/aquamarinepk/aqm v0.0.0
	go.mongodb.org/mongo-driver v1.17.1
)

replace (
	github.com/appetiteclub/appetite/pkg => ../../pkg
	github.com/appetiteclub/appetite/services/kitchen => ../../services/kitchen
	github.com/appetiteclub/appetite/services/order => ../../services/order
	github.com/aquamarinepk/aqm => ../../../aqm
)
