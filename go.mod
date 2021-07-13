module github.com/deanishe/alfred-gcal

require (
	cloud.google.com/go v0.61.0 // indirect
	github.com/deanishe/awgo v0.29.0
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/magefile/mage v1.11.0
	github.com/pkg/errors v0.9.1
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
	google.golang.org/api v0.29.0
	google.golang.org/genproto v0.0.0-20200720141249-1244ee217b7e // indirect
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422

go 1.13
