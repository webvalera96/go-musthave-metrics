package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// ExitCheckAnalyzer проверяет, что в функции main пакета main не используется прямой вызов os.Exit.
//
// Этот анализатор помогает обеспечить правильное завершение программы через возврат из main
// или использование других механизмов завершения (например, через fx.Run() или graceful shutdown),
// что важно для корректной работы тестов и интеграции с системами управления процессами.
var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  runExitCheck,
}

func runExitCheck(pass *analysis.Pass) (interface{}, error) {
	// Проверяем, что мы в пакете main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Проверяем импорт os
		hasOSImport := false
		for _, imp := range file.Imports {
			if imp.Path.Value == `"os"` {
				hasOSImport = true
				break
			}
		}

		if !hasOSImport {
			continue
		}

		// Ищем функцию main
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				return true
			}

			// Проверяем вызовы os.Exit в теле функции main
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Проверяем, является ли это вызовом os.Exit
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				// Проверяем, что это os.Exit
				if ident.Name == "os" && sel.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "прямой вызов os.Exit в функции main пакета main запрещен")
				}

				return true
			})

			return true
		})
	}

	return nil, nil
}
