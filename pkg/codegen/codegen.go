//go:generate go run ${BASE_DIR}/pkg/codegen/cleanup/main.go
//go:generate go run ${BASE_DIR}/pkg/codegen/deepcopy/main.go
//go:generate go run ${BASE_DIR}/pkg/codegen/conversion/main.go
//go:generate go run ${BASE_DIR}/pkg/codegen/defaulter/main.go

package main

func main() {}
