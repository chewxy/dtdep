package main

import (
	"go/types"
	"log"
	"strconv"
	"sync"

	GR "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/simple"
)

const (
	structdep = 1.1
	methdep   = 1.0
)

var atLock = &sync.Mutex{}
var allTypes []T
var uniqTypes = make(map[string]int64)

var graph = make(map[T]map[T]float64) // T -> list of types that depend on T.

func isIgnored(a string) bool {
	for _, s := range ignored {
		if s == a {
			return true
		}
	}
	return false
}

type T struct {
	*types.Named
	id int64
}

// ID implements gonum/graph.Nodee
func (t T) ID() int64     { return t.id }
func (t T) DOTID() string { return fullname(t.Named) }

func fullname(t *types.Named) string {
	if t.Obj().Pkg() == nil {
		return t.Obj().Id()
	}
	return t.Obj().Pkg().Path() + "." + t.Obj().Name()
}

// addKnown adds a known type to the list
func addKnown(t *types.Named) int64 {
	atLock.Lock()
	tid, ok := uniqTypes[fullname(t)]
	if ok {
		atLock.Unlock()
		return tid
	}
	allTypes = append(allTypes, T{Named: t})
	id := int64(len(allTypes) - 1)
	allTypes[id].id = id
	uniqTypes[fullname(t)] = id
	atLock.Unlock()
	return id
}

func add(from, to *types.Named, edgekind float64) {
	// if a dependency is ignored, we forget about it.
	if isIgnored(fullname(to)) {
		if !isIgnored(fullname(from)) {
			addKnown(from)
		}
		return
	}

	// get T of from
	atLock.Lock()
	fromID, ok := uniqTypes[fullname(from)]
	atLock.Unlock()
	if !ok {
		fromID = addKnown(from)
	}
	atLock.Lock()
	fromType := allTypes[fromID]
	atLock.Unlock()

	// get T of to
	atLock.Lock()
	toID, ok := uniqTypes[fullname(to)]
	atLock.Unlock()
	if !ok {
		toID = addKnown(to)
	}
	atLock.Lock()
	toType := allTypes[toID]
	atLock.Unlock()

	// add edge
	c := cost(from, to)
	m, ok := graph[toType]
	if !ok {
		graph[toType] = make(map[T]float64)
		m = graph[toType]
	}

	m[fromType] = c
}

func cost(from, to *types.Named) float64 {
	eF := from.Obj().Exported()
	eT := to.Obj().Exported()

	var toIsClosed bool
	switch tt := to.Underlying().(type) {
	case *types.Interface:
		// if it's an interface:
		// we check whether it's cloesed (i.e. there are at least one unexported method)
		com := tt.Complete()
		for i := 0; i < com.NumMethods(); i++ {
			meth := com.Method(i)
			if !meth.Exported() {
				toIsClosed = true
			}
		}
	}
	switch {
	case eF && eT:
		return 1.0
	case !eT && !toIsClosed:
		return 3.0
	case !eT && toIsClosed:
		return 4 // most heavy
	case !eF:
		return 2.0

	}
	panic("Unreachable")
}

func process(t *types.Named, edgekind float64) {
	switch tt := t.Underlying().(type) {
	case *types.Struct:
		for i := 0; i < tt.NumFields(); i++ {
			ft := tt.Field(i).Type()
			switch ntf := ft.(type) {
			case *types.Named:
				// then add edge t -> ntf
				add(t, ntf, edgekind)
			case *types.Basic:
				continue
			case *types.Pointer:
				n, ok := ntf.Elem().(*types.Named)
				if ok {
					add(t, n, edgekind)
					continue
				}
				log.Printf("%v of *%T unhandled ", tt.Field(i).Name(), ntf.Elem())
			case *types.Slice:
				n, ok := ntf.Elem().(*types.Named)
				if ok {
					add(t, n, edgekind)
				}
			case *types.Map:
				n, ok := ntf.Elem().(*types.Named)
				if ok {
					add(t, n, edgekind)
				}
				n, ok = ntf.Key().(*types.Named)
				if ok {
					add(t, n, edgekind)
				}
			case *types.Chan:
				n, ok := ntf.Elem().(*types.Named)
				if ok {
					add(t, n, edgekind)
				}
			default:
				log.Printf("%v of %T unhandled", tt.Field(i).Name(), ft)
			}
		}
	case *types.Interface:
		for i := 0; i < tt.NumEmbeddeds(); i++ {
			et, ok := tt.EmbeddedType(i).(*types.Named)
			if ok {
				add(t, et, edgekind)
			}
		}
	case *types.Slice:
		n, ok := tt.Elem().(*types.Named)
		if ok {
			add(t, n, edgekind)
		}
	case *types.Signature:
		for i := 0; i < tt.Params().Len(); i++ {
			np, ok := tt.Params().At(i).Type().(*types.Named)
			if ok {
				add(t, np, edgekind)
			}
		}
		for i := 0; i < tt.Results().Len(); i++ {
			nr, ok := tt.Results().At(i).Type().(*types.Named)
			if ok {
				add(t, nr, edgekind)
			}
		}
	default:
		log.Printf("%v of %T unhandled", fullname(t), t.Underlying())
	}
}
func processMethod(f *types.Func) {
	t := f.Type().(*types.Signature)
	if t.Recv() == nil {
		return
	}
	recvType := t.Recv().Type()
	var from *types.Named
	switch rt := recvType.(type) {
	case *types.Pointer:
		n, ok := rt.Elem().(*types.Named)
		if !ok {
			log.Printf("Receiver of %v is not named.", f.FullName())
			return
		}
		from = n
	case *types.Named:
		from = rt
	default:
		log.Printf("receiver type %T not handled", recvType)
	}

	for i := 0; i < t.Params().Len(); i++ {
		np, ok := t.Params().At(i).Type().(*types.Named)
		if ok {
			add(from, np, methdep)
		}
	}
	for i := 0; i < t.Results().Len(); i++ {
		nr, ok := t.Results().At(i).Type().(*types.Named)
		if ok {
			add(from, nr, methdep)
		}
	}
}

type dg struct {
	*simple.WeightedDirectedGraph
}

func newGraph() *simple.WeightedDirectedGraph {
	g := simple.NewWeightedDirectedGraph(5, 0)
	for _, t := range allTypes {
		g.AddNode(t)
	}

	for to, froms := range graph {
		for from, weight := range froms {
			if from.id == to.id {
				continue
			}
			g.SetWeightedEdge(we{g.NewWeightedEdge(from, to, weight)})
		}
	}
	return g
}

type we struct {
	GR.WeightedEdge
}

func (e we) Attributes() []encoding.Attribute {
	color := "blue"
	if e.Weight() == methdep {
		color = "red"
	}
	return []encoding.Attribute{
		{Key: "weight", Value: strconv.FormatFloat(e.Weight(), 'f', -1, 64)},
		{Key: "color", Value: color},
	}
}
