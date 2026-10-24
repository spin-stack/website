package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

// constants are the figures pages state that the code decides, written to src/data/constants.json
// and shown by <Value name="..." />: a page that typed one out was right until the constant moved,
// and nothing said when it did.
type constants struct {
	// RotateAtBytes is the size a volume's top layer commits at whatever its age.
	RotateAtBytes codeValue `json:"rotateAtBytes"`
	// CompatWindow is how much older than its control plane a runner may be.
	CompatWindow codeValue `json:"compatWindow"`
	// LogRetentionDays is how long CloudWatch keeps the flow log and the boot logs by default.
	LogRetentionDays codeValue `json:"logRetentionDays"`
}

// codeValue is one constant: its value, as a page writes it, and where it is declared.
type codeValue struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

const (
	rotatePath = "internal/storage/qcow/manager.go"
	compatPath = "internal/controlplane/compat/compat.go"
	logsModule = "modules/controlplane"
)

// spinConstants is the half of constants.json read from spin.
func spinConstants(c *constants, spin string) error {
	v, err := goConstant(filepath.Join(spin, rotatePath), "DefaultRotateAtBytes")
	if err != nil {
		return err
	}
	n, ok := constant.Int64Val(v)
	if !ok {
		return fmt.Errorf("%s: DefaultRotateAtBytes is no int64", rotatePath)
	}
	c.RotateAtBytes = codeValue{bytesText(n), "spin:" + rotatePath}
	if v, err = goConstant(filepath.Join(spin, compatPath), "Window"); err != nil {
		return err
	}
	if n, ok = constant.Int64Val(v); !ok {
		return fmt.Errorf("%s: Window is no duration", compatPath)
	}
	c.CompatWindow = codeValue{durationText(time.Duration(n)), "spin:" + compatPath}
	return nil
}

// terraformConstants is the half of constants.json read from spin-terraform-aws.
func terraformConstants(c *constants, tf string) error {
	mod, diags := tfconfig.LoadModule(filepath.Join(tf, logsModule))
	if diags.HasErrors() {
		return fmt.Errorf("reading %s: %w", logsModule, diags.Err())
	}
	v, ok := mod.Variables["log_retention_days"]
	if !ok {
		return fmt.Errorf("%s has no variable log_retention_days", logsModule)
	}
	days, ok := v.Default.(float64)
	if !ok {
		return fmt.Errorf("%s: log_retention_days defaults to %v, not a number", logsModule, v.Default)
	}
	rel, err := filepath.Rel(tf, v.Pos.Filename)
	if err != nil {
		return err
	}
	c.LogRetentionDays = codeValue{plural(int64(days), "day"), "spin-terraform-aws:" + filepath.ToSlash(rel)}
	return nil
}

// readConstants is constants.json as the site has it, for -only terraform,brand to keep spin's.
func readConstants(path string) (constants, error) {
	var c constants
	raw, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func (c constants) encode() ([]byte, error) {
	b, err := json.MarshalIndent(c, "", "\t")
	return append(b, '\n'), err
}

// goConstant is the value of the package-level constant name declared in file, worked out as the
// compiler would from the literals, other constants of the file and time's units: reading the
// file, not running it, is what lets the site be built without spin's packages compiling.
func goConstant(file, name string) (constant.Value, error) {
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	decls := map[string]ast.Expr{}
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			vs := s.(*ast.ValueSpec)
			for i, n := range vs.Names {
				if i < len(vs.Values) {
					decls[n.Name] = vs.Values[i]
				}
			}
		}
	}
	e, ok := decls[name]
	if !ok {
		return nil, fmt.Errorf("%s declares no constant %s", file, name)
	}
	v, err := constValue(e, decls, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", file, name, err)
	}
	return v, nil
}

// timeUnits are the time package's constants a duration is written in.
var timeUnits = map[string]time.Duration{
	"Nanosecond": time.Nanosecond, "Microsecond": time.Microsecond, "Millisecond": time.Millisecond,
	"Second": time.Second, "Minute": time.Minute, "Hour": time.Hour,
}

func constValue(e ast.Expr, decls map[string]ast.Expr, depth int) (constant.Value, error) {
	if depth > 16 {
		return nil, fmt.Errorf("constants that name each other")
	}
	switch e := e.(type) {
	case *ast.BasicLit:
		v := constant.MakeFromLiteral(e.Value, e.Kind, 0)
		if v.Kind() == constant.Unknown {
			return nil, fmt.Errorf("the literal %s", e.Value)
		}
		return v, nil
	case *ast.ParenExpr:
		return constValue(e.X, decls, depth)
	case *ast.Ident:
		d, ok := decls[e.Name]
		if !ok {
			return nil, fmt.Errorf("%s is no constant of the file", e.Name)
		}
		return constValue(d, decls, depth+1)
	case *ast.SelectorExpr:
		if pkg, ok := e.X.(*ast.Ident); ok && pkg.Name == "time" {
			if u, ok := timeUnits[e.Sel.Name]; ok {
				return constant.MakeInt64(int64(u)), nil
			}
		}
		return nil, fmt.Errorf("a constant of another package")
	case *ast.BinaryExpr:
		x, err := constValue(e.X, decls, depth)
		if err != nil {
			return nil, err
		}
		y, err := constValue(e.Y, decls, depth)
		if err != nil {
			return nil, err
		}
		switch e.Op {
		case token.SHL, token.SHR:
			s, ok := constant.Uint64Val(y)
			if !ok {
				return nil, fmt.Errorf("a shift by %s", y)
			}
			return constant.Shift(x, e.Op, uint(s)), nil
		case token.ADD, token.SUB, token.MUL, token.QUO:
			op := e.Op
			if op == token.QUO {
				op = token.QUO_ASSIGN // integer division, as for typed integer constants
			}
			return constant.BinaryOp(x, op, y), nil
		}
		return nil, fmt.Errorf("the operator %s", e.Op)
	}
	return nil, fmt.Errorf("an expression the site does not work out: %T", e)
}

// bytesText is a size as a page writes it: in the largest binary unit it is a whole number of.
func bytesText(n int64) string {
	for _, u := range []struct {
		size int64
		name string
	}{{1 << 40, "TiB"}, {1 << 30, "GiB"}, {1 << 20, "MiB"}, {1 << 10, "KiB"}} {
		if n >= u.size && n%u.size == 0 {
			return fmt.Sprintf("%d %s", n/u.size, u.name)
		}
	}
	return plural(n, "byte")
}

// durationText is a duration as a page writes it: in the largest of days, hours, minutes and
// seconds it is a whole number of.
func durationText(d time.Duration) string {
	for _, u := range []struct {
		size time.Duration
		name string
	}{{24 * time.Hour, "day"}, {time.Hour, "hour"}, {time.Minute, "minute"}, {time.Second, "second"}} {
		if d >= u.size && d%u.size == 0 {
			return plural(int64(d/u.size), u.name)
		}
	}
	return d.String()
}

func plural(n int64, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return strconv.FormatInt(n, 10) + " " + unit + "s"
}
