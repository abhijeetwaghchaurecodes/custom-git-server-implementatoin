package validator

import (
	"fmt"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Result holds the Yaegi smoke-test result for one implementation.
type Result struct {
	Valid  bool
	Error  string
}

// Validate uses Yaegi to interpret the Go source and verify the struct symbol exists.
// Yaegi is used as a pre-injection smoke tester — not as the runtime.
// go build is the authoritative validation gate.
func Validate(sourceCode, packageName, structName string) Result {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)

	if _, err := i.Eval(sourceCode); err != nil {
		return Result{false, fmt.Sprintf("yaegi eval: %v", err)}
	}

	v, err := i.Eval(fmt.Sprintf("%s.%s", packageName, structName))
	if err != nil {
		return Result{false, fmt.Sprintf("symbol %s.%s not found: %v", packageName, structName, err)}
	}
	if !v.IsValid() {
		return Result{false, fmt.Sprintf("symbol %s.%s resolved to invalid value", packageName, structName)}
	}
	return Result{Valid: true}
}
