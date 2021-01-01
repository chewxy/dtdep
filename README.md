# dtdep
dtdep is a program that helps you discover data type dependencies within a single package. This allows you to break up packages that are too large by simply doing some graph cutting

# To use:

```
dtdep ignored="_.error,fmt.State,hash.Hash,fmt.Stringer,hash.Hash32" -out=OUTFILE.dot PATH/TO/LIBRARY/TO/BE/ANALYSED
```

Both `ignored` and `out` are optional.
