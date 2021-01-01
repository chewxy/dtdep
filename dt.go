package main

import (
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

var pkgnames []string

func inPkgnames(t *types.Named) bool {
	if t.Obj().Pkg() == nil {
		return false
	}
	for _, n := range pkgnames {
		if t.Obj().Pkg().Path() == n {
			return true
		}
	}
	return false
}

func loadFuncs(pkg *packages.Package) []*types.Func {
	var funcs []*types.Func

	if pkg.TypesInfo == nil {
		log.Printf("%v has no types info\n%#v", pkg, pkg)
		return nil
	}
	for _, obj := range pkg.TypesInfo.Defs {
		if obj, ok := obj.(*types.Func); ok {
			funcs = append(funcs, obj)
		}
	}

	return funcs
}

func loadNamedTypes(pkg *packages.Package) []*types.Named {
	var allNamed []*types.Named

	if pkg.TypesInfo == nil || len(pkg.TypesInfo.Defs) == 0 {
		log.Printf("%v has no types info\n%#v", pkg, pkg)
		return nil
	}
	for _, obj := range pkg.TypesInfo.Defs {
		if obj, ok := obj.(*types.TypeName); ok && !obj.IsAlias() {
			if named, ok := obj.Type().(*types.Named); ok {
				allNamed = append(allNamed, named)
			}
		}
	}

	return allNamed
}
