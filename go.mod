module github.com/deanishe/alfred-gcal

require (
	github.com/deanishe/awgo v0.25.0
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/magefile/mage v1.10.0
	github.com/pkg/errors v0.9.1
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84
	google.golang.org/api v0.43.0
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422

go 1.13
