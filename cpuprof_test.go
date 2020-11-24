package main

//烧机测试，循环执行hardest以获取火焰图

import (
	"bytes"
	"fmt"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

const (
	pprofOutput = "output/pprof"
	pprofDuration = time.Minute
)

func TestCPUProfile(t *testing.T) {
	runtime.GOMAXPROCS(4)
	err := os.MkdirAll(filepath.Dir(pprofOutput), 0755)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(pprofOutput)
	if err != nil {
		t.Fatal(err)
	}

	*flagShowOnlyResult = true
	*flagShowStopAtFirst = true

	hardest, err := ioutil.ReadFile("assets/hardest_1106.txt")
	if err != nil {
		panic(err)
	}
	lines := bytes.Split(hardest, []byte("\n"))

	pprof.StartCPUProfile(f)
	startTime := time.Now()
	for time.Since(startTime) < pprofDuration {
		for _, line := range lines {
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
	pprof.StopCPUProfile()
	outputAbs, err := filepath.Abs(pprofOutput)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "使用这个命令查看结果：\ngo tool pprof -http=:1234 %s\n",
		outputAbs)
}
