# sudoku 数独解题

毫秒级的数独解题，还能显示每一步的推断。

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
    解 1
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
一般难度谜题，如17线索的 puzzles/simple-01.txt 可以在100微妙内完成。
puzzles/hard-02.txt 是某个新闻号称"最难的数独"，本项目找到唯一解的耗时小于1毫秒：

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
    解 1
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
    总耗时：702.759µs
    总猜次数：91

benchmark_test.go 包含测试数据集，本项目解开"HardestDatabase110626" 375 题耗时0.5秒：

    $ go test . -v -count=1 -test.run Hardest1106

## 如何做到 ##

划重点：

- 尽量加强逻辑推理，填充确定的单元格。
- 遇到无确定单元格的局面，选定一个单元格，对多个候选的填充数产生分支，排除矛盾的局面。

### 推理 ###

本项目的核心方法是排除，使用一个9\*9\*9的数组，标记某个单元格是否有可能是某个数。

    //cellExcludes[x][y][n] = 0 ： 单元格(x,y)可能是n
    //cellExcludes[x][y][n] = 1 ： 单元格(x,y)排除n
    cellExcludes [9][9][9]int8    

概念：

- 互斥组：一行、一列或一宫（3\*3）。

符号：

- c : 单元格
- n : 填充数
- B : 互斥组

本项目主要使用下面这些推理：

- 定理1：如果 c 填充 n ，那么 c 排除除 n 外的所有数。

  最直观的一条规则，如果单元格填了一个数，就不能填其他数。

- 定理2：如果 c 排除除 n 外的所有数，那么单元格 c 填充 n。

  与定理 1 互为逆命题，也是直观的规则。

- 定理3：c 属于 B，如果 c 填充 n ，那么 B 内除 c 的单元格排除 n。

  数独的基本规则，是同一行、同一列、同一宫不能填相同的数。每个单元格都属于 3 个互斥组，分别是单元格所在的行、列、宫。
  当单元格填入一个数，其所在 3 个互斥组内所有其他单元格都排除这个数。
    
- 定理4：c 属于 B，如果 B 内除 c 的单元格排除 n，那么 c 填充 n。

  与定理 3 互为逆命题。对应人手解数独的常见推理方法，如果一行内8个单元格都排除一个数，那第9个单元格可以填入这个数。

- 定理5：设单元格集合 C 是两个互斥组 A 和 B 的交集，如果 A-C 内全部单元格排除 n ，那么 B-C 内全部单元格排除 n。

  如以下示意，如果标记 A 的单元格全部排除数 n，显然标记 C 的单元格至少有一个是 n，所以标记 B 全部排除 n。
  
  反过来同理，如果标记 B 全部排除 n，那么标记 A 的单元格全部排除数 n。

      =============================  =============================
       C  C  C | B  B  B | B  B  B            |    A    |         
       A  A  A |         |                    |    A    |                  
       A  A  A |         |                    |    A    |         
      -----------------------------  -----------------------------
               |         |                    | B  C  B |         
               |         |                    | B  C  B |         
               |         |                    | B  C  B |         
      -----------------------------  -----------------------------
               |         |                    |    A    |         
               |         |                    |    A    |         
               |         |                    |    A    |         
      =============================  =============================

  标记 C 的3个单元格称为“核”，任意一宫内的任意一行、列都可以作为一个核，总共有54个核，每一个核形成一对互相排除的区域。


更多的规则可以排除更多不合理选项，减少产生分支。
本项目也曾尝试使用更多的推理规则，但每一条规则都有计算成本，
一些规则开销较大但能减少的分支数很少，使用反而会降低效率。

### 触发式 ###

这是一项工程技巧，相较于使用全局扫描，"触发式"推理法更高效率。"触发式"即一个变量发生改变才去检测它可影响的推理，可以避免循环扫描没有变化的条件。

例如对于推理2，"循环扫描所有单元格排除了除n之外的数"是常规的推理方式，
"触发式"推理实现较复杂：除了要记录每个单元格实际排除的数，还使用 9\*9 的数组统计每个单元格排除了多少个数：

    //cellNumExcludes[x][y] = m ：  单元格(x,y)排除了 m 个数
    //cellNumExcludes[x][y] == SUM(cellExcludes[x][y][...])
    cellNumExcludes [9][9]int8

当执行单元格 (x,y) 排除n：

    func exclude(x,y,n int) {
        if cellExcludes[x][y][n] == 1 { return }
        cellExcludes[x][y][n] = 1
        cellNumExcludes[x][y]++
        if cellNumExcludes[x][y] == 8 {
            //TODO：找到单元格(x,y)未排除的数，并填入
        }
    }

### 分支 ###

分支的另外一种常见说法是“猜”和“回溯”。本项目解题使用递归计算，
“猜”一格时将整个局面复制一份，在发现某种局面矛盾时返回到产生分支的点。具体流程是：

- 开局即开启根分支。
- 一个分支首先使用推理填充所有确定的单元格，直到发生 3 种情况之一：
    - 全部填充完毕，即找到一个解。
    - 发生了矛盾，无法继续演算，即分支没有解。
    - 未填充完毕且没有找到确定的单元格，则选定一个单元格，对多个候选的填充数产生分支，分别递归演算。

生成分支的关键是如何选择产生分支的单元格和猜数的顺序。本项目的选择规则是挑可选项最少的单元格，
多个可选项相同的单元格散列挑选一个（使用散列而不是随机，避免结果随机性），猜数顺序也是散列挑选。
这可能有较大的改进空间。