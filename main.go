// dtdep is a program that draws the dependencies of your datatypes within a package.
package main

import (
	"flag"
	"go/types"
	"io/ioutil"
	"log"
	"strings"

	"golang.org/x/tools/go/packages"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

var ignoredS = flag.String("ignored", "_.error,fmt.State,hash.Hash,fmt.Stringer,hash.Hash32", "what data types to ignore? (comma delimited)")
var outFile = flag.String("out", "foo.dot", "output file name")

var ignored []string

func main() {
	flag.Parse()
	if *ignoredS != "" {
		ignored = strings.Split(*ignoredS, ",")
		log.Printf("%v", ignored)
	}
	dirs := flag.Args()

	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypes | packages.NeedImports | packages.NeedDeps, Tests: false, Dir: dirs[0]}
	ps, err := packages.Load(cfg, dirs...)
	die(err)

	var allNamedTypes []*types.Named
	for _, pkg := range ps {
		pkgnames = append(pkgnames, pkg.ID)
		if pkg.Errors != nil {
			log.Fatalf("%v", pkg.Errors)
		}
		ts := loadNamedTypes(pkg)
		allNamedTypes = append(allNamedTypes, ts...)
	}

	for _, t := range allNamedTypes {
		process(t, structdep)
	}

	var allFuncs []*types.Func
	for _, pkg := range ps {
		fs := loadFuncs(pkg)
		allFuncs = append(allFuncs, fs...)
	}
	for _, f := range allFuncs {
		processMethod(f)
	}

	g := newGraph()
	data, err := dot.Marshal(g, "", "", "")
	die(err)
	ioutil.WriteFile(*outFile, data, 0666)
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}
