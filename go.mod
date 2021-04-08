module github.com/deanishe/alfred-gcal

require (
	github.com/deanishe/awgo v0.25.0
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/magefile/mage v1.10.0
	github.com/pkg/errors v0.9.1
	golang.org/x/net v0.0.0-20210316092652-d523dce5a7f4
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	google.golang.org/api v0.44.0
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422

go 1.13
