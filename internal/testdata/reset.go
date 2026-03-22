package testdata

// generate:reset
type ResetableStruct struct {
	I     int
	Str   string
	StrP  *string
	S     []int
	M     map[string]string
	Child *ResetableStruct
}
