# tfviz

This is a MVP

First project in Golang so don't expect the nicest code.
If project is popular enough, I will be rewriting it for better performances / readability

## Roadmap

TODO

if some of the terraform syntaxes that you use are not yet supported, please open an Issue
more TF objects
more cloud providers

## Test coverage

Can be tested with this:

go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out