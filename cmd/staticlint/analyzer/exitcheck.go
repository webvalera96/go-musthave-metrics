package analyzer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// ExitCheckAnalyzer запрещает:
// - panic — везде
// - os.Exit и log.Fatal — везде, кроме функции main пакета main
var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "запрещает panic везде, а os.Exit/log.Fatal — везде кроме main.main",
	Run:  runExitCheck,
}

func runExitCheck(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}

			isMainFunc := pass.Pkg != nil &&
				pass.Pkg.Name() == "main" &&
				fn.Recv == nil &&
				fn.Name != nil &&
				fn.Name.Name == "main"

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// panic(...) запрещен везде
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
					pass.Reportf(call.Pos(), "вызов panic запрещен")
					return true
				}

				// os.Exit / log.Fatal запрещены везде, кроме main.main
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil {
					return true
				}

				obj := pass.TypesInfo.Uses[sel.Sel]
				fnObj, ok := obj.(*types.Func)
				if !ok || fnObj.Pkg() == nil {
					return true
				}

				switch fnObj.Pkg().Path() {
				case "os":
					if fnObj.Name() == "Exit" && !isMainFunc {
						pass.Reportf(call.Pos(), "вызов os.Exit запрещен вне main.main")
					}
				case "log":
					if fnObj.Name() == "Fatal" && !isMainFunc {
						pass.Reportf(call.Pos(), "вызов log.Fatal запрещен вне main.main")
					}
				}

				return true
			})
		}
	}

	return nil, nil
}
