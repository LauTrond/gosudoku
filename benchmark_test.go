package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func Test17Clue(t *testing.T) {
	benchmark(t, 1, "assets/17_clue.txt", "output/17_clue.txt")
}

func Test17Clue_MT(t *testing.T) {
	benchmark(t, runtime.NumCPU(), "assets/17_clue.txt", "output/17_clue.txt")
}

func TestHardest1905_11(t *testing.T) {
	benchmark(t, 1, "assets/hardest_1905_11.txt", "output/hardest_1905_11.txt")
}

func TestHardest1905_11_MT(t *testing.T) {
	benchmark(t, runtime.NumCPU(), "assets/hardest_1905_11.txt", "output/hardest_1905_11.txt")
}

func TestHardest1106(t *testing.T) {
	benchmark(t, 1, "assets/hardest_1106.txt", "output/hardest_1106.txt")
}

func TestHardest1106_MT(t *testing.T) {
	benchmark(t, runtime.NumCPU(), "assets/hardest_1106.txt", "output/hardest_1106.txt")
}

const overwriteOutput = false

func benchmark(t *testing.T, parallel int, inputFile, outputFile string) {
	if parallel < 1 {
		t.Fatal("parallel must >= 1")
	}

	runtime.GOMAXPROCS(parallel + 2)
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	*flagShowOnlyResult = true
	//*flagStopAtFirstSolution = true

	input, err := os.Open(inputFile)
	check(err)
	defer input.Close()
	br := bufio.NewReader(input)

	if !overwriteOutput {
		if _, err = os.Stat(outputFile); err == nil {
			//outputFile exists
			fmt.Printf("%s 文件已经存在，屏蔽输出\n", outputFile)
			outputFile = ""
		}
	}

	var output *os.File
	if outputFile != "" {
		outputDir := filepath.Dir(outputFile)
		outputTmp := "." + filepath.Base(outputFile) + ".tmp"
		outputTmpPath := filepath.Join(outputDir, outputTmp)
		err = os.MkdirAll(outputDir, 0755)
		check(err)
		output, err = os.Create(outputTmpPath)
		check(err)
		defer func() {
			err := output.Close()
			check(err)
			err = os.Rename(outputTmpPath, outputFile)
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
	outputFilePrint := outputFile
	if outputFilePrint == "" {
		outputFilePrint = "<无>"
	}
	fmt.Printf("测试集：%v\n", inputFile)
	fmt.Printf("输出文件：%s\n", outputFilePrint)
	fmt.Printf("线程数：%v\n", parallel)
	fmt.Printf("启动时间：%v\n", startTime.Format("2006-01-02 15:04:05"))

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

	if parallel == 1 {
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
		outputChannels := make(chan chan []byte, parallel*1024)
		throttle := make(chan struct{}, parallel)

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
	fmt.Printf("总耗时：%.3fs\n", dur.Seconds())
	fmt.Printf("总局数：%d\n", puzzlesCount)
	fmt.Printf("唯一解局数：%d\n", succCount)
	fmt.Printf("速率(局/s)：%.2f\n", float64(puzzlesCount)/dur.Seconds())
	fmt.Printf("总猜次数：%d\n", guessesCount)
	fmt.Printf("总演算次数：%d\n", evalCount)
}
