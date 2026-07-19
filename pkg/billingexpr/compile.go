package billingexpr

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
)

const maxCacheSize = 256

// DefaultExprVersion is used when an expression string has no version prefix.
const DefaultExprVersion = 1

// ParseExprVersion extracts the version tag and body from an expression string.
// Format: "v1:tier(...)" → version=1, body="tier(...)".
// No prefix defaults to DefaultExprVersion.
func ParseExprVersion(exprStr string) (version int, body string) {
	separator := strings.IndexByte(exprStr, ':')
	if separator > 1 && exprStr[0] == 'v' {
		parsedVersion, err := strconv.Atoi(exprStr[1:separator])
		if err == nil {
			return parsedVersion, exprStr[separator+1:]
		}
	}
	return DefaultExprVersion, exprStr
}

type cachedEntry struct {
	prog     *vm.Program
	usedVars map[string]bool
	version  int
}

type RequestReferences struct {
	Headers []string
	Params  []string
}

var (
	cacheMu sync.RWMutex
	cache   = make(map[string]*cachedEntry, 64)
)

// compileEnvPrototypeV1 is the v1 type-checking prototype used at compile time.
var compileEnvPrototypeV1 = map[string]interface{}{
	"p":       float64(0),
	"c":       float64(0),
	"len":     float64(0),
	"cr":      float64(0),
	"cc":      float64(0),
	"cc1h":    float64(0),
	"img":     float64(0),
	"img_o":   float64(0),
	"ai":      float64(0),
	"ao":      float64(0),
	"tier":    func(string, float64) float64 { return 0 },
	"header":  func(string) string { return "" },
	"param":   func(string) interface{} { return nil },
	"has":     func(interface{}, string) bool { return false },
	"hour":    func(string) int { return 0 },
	"minute":  func(string) int { return 0 },
	"weekday": func(string) int { return 0 },
	"month":   func(string) int { return 0 },
	"day":     func(string) int { return 0 },
	"max":     math.Max,
	"min":     math.Min,
	"abs":     math.Abs,
	"ceil":    math.Ceil,
	"floor":   math.Floor,
}

var compileEnvPrototypeV2 = func() map[string]interface{} {
	env := make(map[string]interface{}, len(compileEnvPrototypeV1)+9)
	for key, value := range compileEnvPrototypeV1 {
		env[key] = value
	}
	env["units"] = float64(0)
	env["seconds"] = float64(0)
	env["width"] = float64(0)
	env["height"] = float64(0)
	env["quality"] = ""
	env["resolution_tier"] = ""
	env["image_size_tier"] = ""
	env["image_size"] = ""
	env["usd"] = func(float64) float64 { return 0 }
	return env
}()

func getCompileEnv(version int) map[string]interface{} {
	switch version {
	case 2:
		return compileEnvPrototypeV2
	default:
		return compileEnvPrototypeV1
	}
}

// CompileFromCache compiles an expression string, using a cached program when
// available. The cache is keyed by the SHA-256 hex digest of the expression.
func CompileFromCache(exprStr string) (*vm.Program, error) {
	return compileFromCacheByHash(exprStr, ExprHashString(exprStr))
}

// CompileFromCacheByHash is like CompileFromCache but accepts a pre-computed
// hash, useful when the caller already has the BillingSnapshot.ExprHash.
func CompileFromCacheByHash(exprStr, hash string) (*vm.Program, error) {
	return compileFromCacheByHash(exprStr, hash)
}

func compileFromCacheByHash(exprStr, hash string) (*vm.Program, error) {
	cacheMu.RLock()
	if entry, ok := cache[hash]; ok {
		cacheMu.RUnlock()
		return entry.prog, nil
	}
	cacheMu.RUnlock()

	version, body := ParseExprVersion(exprStr)
	if version != 1 && version != 2 {
		return nil, fmt.Errorf("unsupported billing expression version v%d", version)
	}
	prog, err := expr.Compile(body, expr.Env(getCompileEnv(version)), expr.AsFloat64())
	if err != nil {
		return nil, fmt.Errorf("expr compile error: %w", err)
	}

	vars := extractUsedVars(prog)

	cacheMu.Lock()
	if len(cache) >= maxCacheSize {
		cache = make(map[string]*cachedEntry, 64)
	}
	cache[hash] = &cachedEntry{prog: prog, usedVars: vars, version: version}
	cacheMu.Unlock()

	return prog, nil
}

// ExprVersion returns the version of a cached expression. Returns DefaultExprVersion
// if the expression hasn't been compiled yet or is empty.
func ExprVersion(exprStr string) int {
	if exprStr == "" {
		return DefaultExprVersion
	}
	hash := ExprHashString(exprStr)
	cacheMu.RLock()
	if entry, ok := cache[hash]; ok {
		cacheMu.RUnlock()
		return entry.version
	}
	cacheMu.RUnlock()
	v, _ := ParseExprVersion(exprStr)
	return v
}

func extractUsedVars(prog *vm.Program) map[string]bool {
	vars := make(map[string]bool)
	node := prog.Node()
	ast.Find(node, func(n ast.Node) bool {
		if id, ok := n.(*ast.IdentifierNode); ok {
			vars[id.Value] = true
		}
		return false
	})
	return vars
}

// UsedVars returns the set of identifier names referenced by an expression.
// The result is cached alongside the compiled program. Returns nil for empty input.
func UsedVars(exprStr string) map[string]bool {
	if exprStr == "" {
		return nil
	}
	hash := ExprHashString(exprStr)
	cacheMu.RLock()
	if entry, ok := cache[hash]; ok {
		cacheMu.RUnlock()
		return entry.usedVars
	}
	cacheMu.RUnlock()

	// Compile (and cache) to populate usedVars
	if _, err := compileFromCacheByHash(exprStr, hash); err != nil {
		return nil
	}
	cacheMu.RLock()
	entry, ok := cache[hash]
	cacheMu.RUnlock()
	if ok {
		return entry.usedVars
	}
	return nil
}

// ReferencedRequestInput returns the literal header names and JSON paths read
// by header() and param(). Async billing snapshots require literal arguments so
// they can persist only the values needed for settlement.
func ReferencedRequestInput(exprStr string) (RequestReferences, error) {
	prog, err := CompileFromCache(exprStr)
	if err != nil {
		return RequestReferences{}, err
	}

	headerSet := make(map[string]struct{})
	paramSet := make(map[string]struct{})
	var referenceErr error
	ast.Find(prog.Node(), func(n ast.Node) bool {
		call, ok := n.(*ast.CallNode)
		if !ok {
			return false
		}
		callee, ok := call.Callee.(*ast.IdentifierNode)
		if !ok || (callee.Value != "header" && callee.Value != "param") {
			return false
		}
		if len(call.Arguments) != 1 {
			referenceErr = fmt.Errorf("%s() must have exactly one literal string argument", callee.Value)
			return true
		}
		argument, ok := call.Arguments[0].(*ast.StringNode)
		if !ok {
			referenceErr = fmt.Errorf("%s() must use a literal string argument for async settlement", callee.Value)
			return true
		}
		value := strings.TrimSpace(argument.Value)
		if value == "" {
			referenceErr = fmt.Errorf("%s() argument must not be empty", callee.Value)
			return true
		}
		if callee.Value == "header" {
			headerSet[strings.ToLower(value)] = struct{}{}
		} else {
			paramSet[value] = struct{}{}
		}
		return false
	})
	if referenceErr != nil {
		return RequestReferences{}, referenceErr
	}

	references := RequestReferences{
		Headers: make([]string, 0, len(headerSet)),
		Params:  make([]string, 0, len(paramSet)),
	}
	for header := range headerSet {
		references.Headers = append(references.Headers, header)
	}
	for path := range paramSet {
		references.Params = append(references.Params, path)
	}
	sort.Strings(references.Headers)
	sort.Strings(references.Params)
	return references, nil
}

// InvalidateCache clears the compiled-expression cache.
// Called when billing rules are updated.
func InvalidateCache() {
	cacheMu.Lock()
	cache = make(map[string]*cachedEntry, 64)
	cacheMu.Unlock()
}
