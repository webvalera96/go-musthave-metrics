// Собственный анализатор:
//   - exitcheck: запрещает panic везде, а log.Fatal/os.Exit — везде кроме main.main
package analyzer

import (
	"golang.org/x/tools/go/analysis"
)

// Analyzer — единственный анализатор для singlechecker.
var Analyzer *analysis.Analyzer = ExitCheckAnalyzer
