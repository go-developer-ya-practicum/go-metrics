// Package exitcallanalizer содержит статический анализатор,
// запрещающий использовать прямой вызов os.Exit в функции main пакета main
package exitcallanalizer

import (
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var ExitCallAnalyzer = &analysis.Analyzer{
	Name: "exit_call",
	Doc:  "checks call of os.Exit in function main of package main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		pkgName := file.Name.Name
		path := pass.Fset.Position(file.Pos()).Filename
		filename := filepath.Base(path)

		if pkgName != "main" || !strings.HasSuffix(filename, ".go") {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncDecl:
				if x.Name.Name != "main" {
					return false
				}
			case *ast.CallExpr:
				if selExpr, ok := x.Fun.(*ast.SelectorExpr); ok {
					if expr, ok := selExpr.X.(*ast.Ident); ok {
						if expr.Name == "os" && selExpr.Sel.Name == "Exit" {
							pass.Reportf(node.Pos(), "call os.Exit() in main function of main package")
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
