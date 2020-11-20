# sudoku 数独解题

微秒级的数独解题，还能显示每一步的推断。

这个项目的重点是研究数独高效解题方法。

## 如何使用 ##

试试：

    $ go get github.com/LauTrond/sudoku
    $ echo "
    ...7.....
    1........
    ...43.2..
    ........6
    ...5.9...
    ......418
    ....81...
    ..2....5.
    .4....3..
    " | go run github.com/LauTrond/sudoku
    
    =============================
    <17> 开始
             | 7       |         
     1       |         |         
             | 4  3    | 2       
    -----------------------------
             |         |       6 
             | 5     9 |         
             |         | 4  1  8 
    -----------------------------
             |    8  1 |         
           2 |         |    5    
        4    |         | 3       
    =============================
    <18> 该行唯一可以填 1 的位置
             | 7       |         
     1       |         |         
             | 4  3    | 2    [1]
    -----------------------------
             |         |       6 
             | 5     9 |         
             |         | 4  1  8 
    -----------------------------
             |    8  1 |         
           2 |         |    5    
        4    |         | 3       
    =============================
    <19> 该行唯一可以填 1 的位置
         该宫唯一可以填 1 的位置
             | 7 [1]   |         
     1       |         |         
             | 4  3    | 2     1 
    -----------------------------
             |         |       6 
             | 5     9 |         
             |         | 4  1  8 
    -----------------------------
             |    8  1 |         
           2 |         |    5    
        4    |         | 3       

    ....
    
    找到了 1 个解
    =============================
    result 0
     2  6  4 | 7  1  5 | 8  3  9 
     1  3  7 | 8  9  2 | 6  4  5 
     5  9  8 | 4  3  6 | 2  7  1 
    -----------------------------
     4  2  3 | 1  7  8 | 5  9  6 
     8  1  6 | 5  4  9 | 7  2  3 
     7  5  9 | 6  2  3 | 4  1  8 
    -----------------------------
     3  7  5 | 2  8  1 | 9  6  4 
     9  8  2 | 3  6  4 | 1  5  7 
     6  4  1 | 9  5  7 | 3  8  2 

谜题写在txt文件中，可以作为参数来运行：

    $ cd $GOPATH/src/github.com/LauTrond/sudoku
    $ go run . puzzles/simple-01.txt

使用 -h 参数显示命令选项：

    $ cd $GOPATH/src/github.com/LauTrond/sudoku
    $ go run . -h

使用 -b 参数可以显示运算耗时（不包含程序启动、输入输出时间）。
一般难度谜题，如17线索的 puzzles/simple-1.txt 可以在100微妙内完成。
puzzles/hard-02.txt 是某个新闻号称"最难的数独"，本项目找到唯一解的耗时约1毫秒：

    $ cd $GOPATH/src/github.com/LauTrond/sudoku
    $ cat puzzles/hard-02.txt
    8........
    ..36.....
    .7..9.2..
    .5...7...
    ....457..
    ...1...3.
    ..1....68
    ..85...1.
    .9....4..
    
    $ go run . -b puzzles/hard-02.txt
      
    找到了 1 个解
    =============================
    result 0
     8  1  2 | 7  5  3 | 6  4  9 
     9  4  3 | 6  8  2 | 1  7  5 
     6  7  5 | 4  9  1 | 2  8  3 
    -----------------------------
     1  5  4 | 2  3  7 | 8  9  6 
     3  6  9 | 8  4  5 | 7  2  1 
     2  8  7 | 1  6  9 | 5  3  4 
    -----------------------------
     5  2  1 | 9  7  4 | 3  6  8 
     4  3  8 | 5  2  6 | 9  1  7 
     7  9  6 | 3  1  8 | 4  5  2 
    
    总耗时：1.106528ms
    总推演次数：251

但对于下面这些题都是小儿科。文件 hardest_test.go 内包含一个高难数独题集合 "HardestDatabase110626"，全部375题耗时不足2秒：

    $ go test . -v -count=1 Hardest

最难的一道题耗时 40 毫秒进行了 13891 次推理才找到唯一的解。

## 如何做到 ##

划重点：

- 尽量加强逻辑推理，填充确定的单元格。
- 遇到无确定单元格的局面，选定一个单元格，对多个候选的填充数产生分支，排除矛盾的局面。

### 推理 ###

概念：

- 互斥组：一行、一列或一宫（3*3）内一个数字不能出现超过一次，所以称为一个互斥组。

符号：

- c : 单元格
- n : 填充数
- B : 互斥组

本项目主要使用下面这些推理：

- 推理1：如果 c 填充 n ，那么 c 排除除 n 外的所有数。
- 推理2：如果 c 排除除 n 外的所有数，那么单元格 c 填充 n。
- 推理3：如果 c 填充 n ，那么包含 c 的互斥组内除 c 的单元格排除 n。
    - 每个单元格都属于 3 个互斥组，分别是单元格所在的行、列、宫，3 个互斥组都需要排除。  
- 推理4：如果某个包含 c 的互斥组除 c 的单元格排除 n，那么单元格 c 填充 n。

另外在以上常规推理无法找到确定的单元格时，也会使用以下更大开销的规则：

- 规则1（占位排除法）：设单元格集合 C 是两个互斥组 B1 和 B2 的交集，如果 B1-C 内全部单元格排除 n ，那么 B2-C 内全部单元格排除 n。
- 规则2（XWing排除法）：如果 n 在行 R1 和 R2 有且只有 2 个可能的位置，而且两个位置属于相同的列 C1 和 C2，那么 C1 和 C2 内除 R1 和 R2 的所有单元格排除排除 n。行列置换也适用本规则。

### 分支 ###

- 开局即开启根分支。
- 一个分支首先使用推理填充所有确定的单元格，直到发生 3 种情况之一：
    - 全部填充完毕，即找到一个解。
    - 发生了矛盾，无法继续演算，即分支没有解。
    - 未填充完毕且没有找到确定的单元格，则选定一个单元格，对多个候选的填充数产生分支，分别递归演算。
