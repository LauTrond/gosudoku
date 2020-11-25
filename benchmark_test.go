package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func Test17Clue(t *testing.T) {
	(&BenchmarkConfig{
		InputFile: "assets/17_clue.txt",
		OutputFile: "output/17_clue.txt",
	}).Run(t)
}

func Test17Clue_MT(t *testing.T) {
	(&BenchmarkConfig{
		InputFile: "assets/17_clue.txt",
		OutputFile: "output/17_clue.txt",
		Parallel: runtime.NumCPU(),
	}).Run(t)
}

func TestHardest1905_11(t *testing.T) {
	(&BenchmarkConfig{
		InputFile: "assets/hardest_1905_11.txt",
		OutputFile: "output/hardest_1905_11.txt",
	}).Run(t)
}

func TestHardest1905_11_MT(t *testing.T) {
	(&BenchmarkConfig{
		InputFile: "assets/hardest_1905_11.txt",
		OutputFile: "output/hardest_1905_11.txt",
		Parallel: runtime.NumCPU(),
	}).Run(t)
}

func TestHardest1106(t *testing.T) {
	(&BenchmarkConfig{
		InputFile: "assets/hardest_1106.txt",
		OutputFile: "output/hardest_1106.txt",
	}).Run(t)
}

func TestHardest1106_MT(t *testing.T) {
	(&BenchmarkConfig{
		InputFile: "assets/hardest_1106.txt",
		OutputFile: "output/hardest_1106.txt",
		Parallel: runtime.NumCPU(),
	}).Run(t)
}

type BenchmarkConfig struct {
	Parallel int
	InputFile string
	OutputFile string
	OverwriteOutputFile bool
}

func (cfg *BenchmarkConfig) Run(t *testing.T) {
	if cfg.Parallel < 1 {
		cfg.Parallel = 1
	}

	runtime.GOMAXPROCS(cfg.Parallel + 2)
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	*flagShowOnlyResult = true
	//*flagStopAtFirstSolution = true

	input, err := os.Open(cfg.InputFile)
	check(err)
	defer input.Close()
	br := bufio.NewReader(input)

	if !cfg.OverwriteOutputFile {
		if _, err = os.Stat(cfg.OutputFile); err == nil {
			//outputFile exists
			fmt.Printf("%s 文件已经存在，屏蔽输出\n", cfg.OutputFile)
			cfg.OutputFile = ""
		}
	}

	var output *os.File
	if cfg.OutputFile != "" {
		outputDir := filepath.Dir(cfg.OutputFile)
		outputTmp := "." + filepath.Base(cfg.OutputFile) + ".tmp"
		outputTmpPath := filepath.Join(outputDir, outputTmp)
		err = os.MkdirAll(outputDir, 0755)
		check(err)
		output, err = os.Create(outputTmpPath)
		check(err)
		defer func() {
			err := output.Close()
			check(err)
			err = os.Rename(outputTmpPath, cfg.OutputFile)
			check(err)
		}()
	} else {
		output, err = os.Create(os.DevNull)
		check(err)
		defer func() {
			err := output.Close()
			check(err)
		}()
	}

	var mtx sync.Mutex
	puzzlesCount := 0
	guessesCount := 0
	evalCount := 0
	succCount := 0

	startTime := time.Now()
	outputFilePrint := cfg.OutputFile
	if outputFilePrint == "" {
		outputFilePrint = "<无>"
	}
	printNamedValue("测试集", "%s", cfg.InputFile)
	printNamedValue("输出文件","%s", outputFilePrint)
	printNamedValue("线程数","%d", cfg.Parallel)
	printNamedValue("启动时间","%s", startTime.Format("2006-01-02 15:04:05"))

	getLine := func() ([]byte, bool) {
		for {
			line, err := br.ReadBytes('\n')
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			if len(line) == 0 {
				return nil, false
			}
			line = bytes.TrimSuffix(line, []byte("\n"))
			if len(line) != 81 {
				continue
			}
			return line, true
		}
	}

	proceed := func(line []byte) []byte {
		s, trg := ParseSituationFromLine(line)
		ctx := newSudokuContext()
		ctx.Run(s, trg)

		var solutionLine []byte

		if len(ctx.solutions) == 1 {
			solution := ctx.solutions[0]
			solutionLine = make([]byte, 82)
			for r := range loop9 {
				for c := range loop9 {
					solutionLine[r*9+c] = byte('1' + solution[r][c])
				}
			}
			solutionLine[81] = '\n'
		} else {
			solutionLine = []byte(fmt.Sprintf("%d solution(s)", len(ctx.solutions)))
		}

		mtx.Lock()
		puzzlesCount += 1
		if len(ctx.solutions) == 1 {
			succCount++
		}
		guessesCount += ctx.guessesCount
		evalCount += ctx.evalCount
		mtx.Unlock()
		s.Release()

		return solutionLine
	}

	if cfg.Parallel == 1 {
		for {
			puzzleLine, ok := getLine()
			if !ok {
				break
			}
			resultLine := proceed(puzzleLine)
			_, err = output.Write(resultLine)
			check(err)
		}
	} else {
		outputChannels := make(chan chan []byte, cfg.Parallel*1024)
		throttle := make(chan struct{}, cfg.Parallel)

		go func() {
			for {
				puzzleLine, ok := getLine()
				if !ok {
					break
				}
				throttle <- struct{}{}
				lineChan := make(chan []byte, 1)
				outputChannels <- lineChan
				go func() {
					defer func() { <-throttle }()
					resultBytes := proceed(puzzleLine)
					lineChan <- resultBytes
				}()
			}
			close(outputChannels)
		}()

		for {
			c, ok := <-outputChannels
			if !ok {
				break
			}
			_, err = output.Write(<-c)
			check(err)
		}
	}

	dur := time.Since(startTime)
	printNamedValue("总耗时(s)", "%.3f", dur.Seconds())
	printNamedValue("总局数", "%d", puzzlesCount)
	printNamedValue("唯一解局数","%d", succCount)
	printNamedValue("解题速率(局/s)","%.2f", float64(puzzlesCount)/dur.Seconds())
	printNamedValue("分支数","%d", guessesCount)
	printNamedValue("分支率(次/局)","%.2f", float64(guessesCount)/float64(puzzlesCount))
	printNamedValue("总演算次数", "%d", evalCount)
}

func printNamedValue(name string, valueFmt string, value interface{}) {
	tab := strings.Repeat(" ", 15-textWidth(name))
	fmt.Printf("%s:%s%s\n", name, tab, fmt.Sprintf(valueFmt, value))
}

func textWidth(text string) int {
	w := 0
	for _, r := range text {
		if r > 127 {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}