package data

import _ "embed"

//go:embed summary.json
var SummaryJSON []byte

//go:embed AFINN-zh_cn.json
var AFINNJSON []byte

//go:embed user-words.txt
var UserWordsTXT string

//go:embed D_U_protection.json
var DUProtectionJSON []byte
