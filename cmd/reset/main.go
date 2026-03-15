package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/webvalera96/go-musthave-metrics/cmd/reset/generator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <root-directory>\n", os.Args[0])
		os.Exit(1)
	}

	rootDir := os.Args[1]
	if err := processDirectory(rootDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func processDirectory(rootDir string) error {
	packages := make(map[string]*generator.PackageInfo)

	// Сканируем все директории
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем служебные директории
		if info.IsDir() {
			// Пропускаем только скрытые директории и vendor, но не cmd (чтобы можно было запускать из корня)
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			if info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Обрабатываем только .go файлы (кроме сгенерированных)
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".gen.go") {
			return nil
		}

		return processFile(path, packages)
	})

	if err != nil {
		return err
	}

	// Генерируем файлы reset.gen.go для каждого пакета
	for pkgPath, pkgInfo := range packages {
		if len(pkgInfo.Structs) > 0 {
			if err := generator.GenerateResetFile(pkgPath, pkgInfo); err != nil {
				return fmt.Errorf("failed to generate reset file for %s: %w", pkgPath, err)
			}
		}
	}

	return nil
}

func processFile(filePath string, packages map[string]*generator.PackageInfo) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		// Игнорируем ошибки парсинга (например, синтаксические ошибки)
		return nil
	}

	// Получаем путь к пакету
	dir := filepath.Dir(filePath)
	pkgName := node.Name.Name

	if packages[dir] == nil {
		packages[dir] = &generator.PackageInfo{
			Name:    pkgName,
			Structs: make([]*generator.StructInfo, 0),
		}
	}

	pkgInfo := packages[dir]

	// Ищем структуры с комментарием // generate:reset
	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		// Проверяем комментарии перед объявлением
		if genDecl.Doc == nil {
			return true
		}

		hasResetComment := false
		for _, comment := range genDecl.Doc.List {
			if strings.Contains(comment.Text, "generate:reset") {
				hasResetComment = true
				break
			}
		}

		if !hasResetComment {
			return true
		}

		// Обрабатываем все типы в объявлении
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			structInfo := &generator.StructInfo{
				Name:   typeSpec.Name.Name,
				Fields: make([]*generator.FieldInfo, 0),
			}

			// Обрабатываем поля структуры
			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					fieldType := generator.GetFieldType(field.Type)
					for _, name := range field.Names {
						structInfo.Fields = append(structInfo.Fields, &generator.FieldInfo{
							Name: name.Name,
							Type: fieldType,
						})
					}
					// Анонимные поля
					if len(field.Names) == 0 {
						structInfo.Fields = append(structInfo.Fields, &generator.FieldInfo{
							Name: "",
							Type: fieldType,
						})
					}
				}
			}

			pkgInfo.Structs = append(pkgInfo.Structs, structInfo)
		}

		return true
	})

	return nil
}
