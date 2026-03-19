package main

import (
	"github.com/webvalera96/go-musthave-metrics/cmd/staticlint/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
