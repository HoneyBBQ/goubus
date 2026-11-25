package main

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryExportedFieldAndMethodIsDocumented checks that every exported type
// field and method in public packages of the module has a GoDoc comment.
func TestEveryExportedFieldAndMethodIsDocumented(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	for _, dir := range publicPackageDirs(t, repoRoot) {
		for _, missing := range undocumented(t, repoRoot, dir) {
			t.Errorf("%s: %s has no doc comment", dir, missing)
		}
	}
}

func publicPackageDirs(t *testing.T, repoRoot string) []string {
	t.Helper()
	var dirs []string
	err := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		name := entry.Name()
		if rel != "." && (name == "internal" || name == "testdata" ||
			name == "api" || name == "ci" || name == "docs" || name == "scripts" || strings.HasPrefix(name, ".")) {
			return filepath.SkipDir
		}
		matches, globErr := filepath.Glob(filepath.Join(path, "*.go"))
		if globErr != nil {
			return globErr
		}
		if len(matches) > 0 {
			dirs = append(dirs, rel)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	return dirs
}

func undocumented(t *testing.T, repoRoot, relDir string) []string {
	t.Helper()
	fullDir := filepath.Join(repoRoot, relDir)
	paths, err := filepath.Glob(filepath.Join(fullDir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", relDir, err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", path, parseErr)
		}
		files = append(files, file)
	}
	docPkg, err := doc.NewFromFiles(fset, files, "github.com/honeybbq/goubus/v2/"+relDir)
	if err != nil {
		t.Fatalf("doc %s: %v", relDir, err)
	}
	missing := make([]string, 0, len(docPkg.Types))
	for _, typ := range docPkg.Types {
		missing = append(missing, undocumentedMembers(typ)...)
	}

	return missing
}

func undocumentedMembers(typ *doc.Type) []string {
	var missing []string
	for _, spec := range typ.Decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		switch definition := typeSpec.Type.(type) {
		case *ast.StructType:
			missing = append(missing, undocumentedFields(typ.Name, definition.Fields)...)
		case *ast.InterfaceType:
			missing = append(missing, undocumentedFields(typ.Name, definition.Methods)...)
		}
	}
	for _, method := range typ.Methods {
		if ast.IsExported(method.Name) && strings.TrimSpace(method.Doc) == "" {
			missing = append(missing, "method "+typ.Name+"."+method.Name)
		}
	}

	return missing
}

func undocumentedFields(typeName string, fields *ast.FieldList) []string {
	var missing []string
	for _, field := range fields.List {
		if len(field.Names) == 0 || field.Doc != nil || field.Comment != nil {
			continue
		}
		for _, name := range field.Names {
			if name.IsExported() {
				missing = append(missing, typeName+"."+name.Name)
			}
		}
	}

	return missing
}
