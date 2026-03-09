package main

import (
	"github.com/webvalera96/go-musthave-metrics/cmd/staticlint/analyzer"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(analyzer.GetAnalyzers()...)
}
