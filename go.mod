module github.com/deanishe/alfred-gcal

require (
	cloud.google.com/go v0.51.0 // indirect
	github.com/bmatcuk/doublestar v1.2.2
	github.com/deanishe/awgo v0.22.0
	github.com/disintegration/imaging v1.6.2
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/magefile/mage v1.9.0
	github.com/pkg/errors v0.8.1
	golang.org/x/image v0.0.0-20191214001246-9130b4cfad52 // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50 // indirect
	google.golang.org/api v0.23.0
	google.golang.org/genproto v0.0.0-20200108215221-bd8f9a0ef82f // indirect
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422

go 1.13
