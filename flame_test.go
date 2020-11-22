package main

//烧机测试，循环执行hardest以获取火焰图

import (
	"bytes"
	"net/http"
	_ "net/http/pprof"
	"testing"
)

func TestFlame(t *testing.T) {
	//go tool pprof -http=:1234 http://localhost:19190/debug/pprof/profile
	go http.ListenAndServe("localhost:19190", nil)

	*flagShowOnlyResult = true
	*flagShowStopAtFirst = true

	for {
		for _, line := range bytes.Split([]byte(hardest), []byte("\n")) {
			puzzle := bytes.SplitN(line, []byte(","), 2)[0]
			if len(puzzle) != 81 {
				continue
			}

			s, trg := ParseSituationFromLine(puzzle)

			ctx := newSudokuContext()
			ctx.Run(s, trg)
			s.Release()
		}
	}
}
