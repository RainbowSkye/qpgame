package alg

import "fmt"

type Table struct {
	keyDic        map[string]bool
	keyGuiDic     map[int]map[string]bool
	keyFengDic    map[string]bool
	keyFengGuiDic map[int]map[string]bool
}

func NewTable() *Table {
	t := &Table{
		keyDic:        make(map[string]bool),
		keyGuiDic:     make(map[int]map[string]bool),
		keyFengDic:    make(map[string]bool),
		keyFengGuiDic: make(map[int]map[string]bool),
	}
	t.gen()
	return t
}

// 生成胡牌字典
func (t *Table) gen() {
	cards := []int{0, 0, 0, 0, 0, 0, 0, 0, 0}
	level := 0
	feng := false
	jiang := false
	t.genTableNoGui(cards, level, feng, jiang)
	t.genTableGui(feng)
	feng = true
	t.genTableNoGui(cards, level, feng, jiang)
	t.genTableGui(feng)
	fmt.Println(t.keyDic)
	fmt.Println(t.keyGuiDic)
	fmt.Println(t.keyFengDic)
	fmt.Println(t.keyFengGuiDic)
}

func (t *Table) genTableNoGui(cards []int, level int, feng bool, jiang bool) {
	for i := 0; i < 9; i++ {
		if feng && i > 6 {
			continue
		}
		// 1. 需要先将cards中的牌数量计算出来，后续做判断用
		totalCardCount := t.calcTotalCardCount(cards)
		// 加刻子 AAA
		if totalCardCount <= 11 && cards[i] <= 1 {
			cards[i] += 3
			key := t.genKey(cards)
			if feng {
				t.keyFengDic[key] = true
			} else {
				t.keyDic[key] = true

			}
			if level < 5 {
				t.genTableNoGui(cards, level+1, feng, jiang)
			}
			cards[i] -= 3
		}
		// 加连子 ABC
		if totalCardCount <= 11 && i < 7 && cards[i] <= 3 && cards[i+1] <= 3 && cards[i+2] <= 3 {
			cards[i]++
			cards[i+1]++
			cards[i+2]++
			key := t.genKey(cards)
			if feng {
				t.keyFengDic[key] = true
			} else {
				t.keyDic[key] = true

			}
			if level < 5 {
				t.genTableNoGui(cards, level+1, feng, jiang)
			}
			cards[i]--
			cards[i+1]--
			cards[i+2]--
		}
		// 加将 AA
		if !jiang && totalCardCount <= 12 && cards[i] <= 2 {
			cards[i] += 2
			key := t.genKey(cards)
			if feng {
				t.keyFengDic[key] = true
			} else {
				t.keyDic[key] = true

			}
			if level < 5 {
				t.genTableNoGui(cards, level+1, feng, true)
			}
			cards[i] -= 2
		}
	}
}

func (t *Table) calcTotalCardCount(cards []int) int {
	count := 0
	for _, v := range cards {
		count += v
	}
	return count
}

func (t *Table) genKey(cards []int) string {
	key := ""
	dict := []string{"0", "1", "2", "3", "4"}
	for _, v := range cards {
		key += dict[v]
	}
	return key
}

// 生成含鬼牌的赢牌字典
// 从无鬼牌的赢牌字典中替换
func (t *Table) genTableGui(feng bool) {
	dict := t.keyDic
	if feng {
		dict = t.keyFengDic
	}
	for key := range dict {
		cards := t.toNumberArray(key)
		t.genGui(cards, 1, feng)
	}
}

func (t *Table) toNumberArray(str string) []int {
	strByte := []byte(str)
	cards := make([]int, len(strByte))
	for i, v := range strByte {
		cards[i] = int(v)
	}
	return cards
}

func (t *Table) genGui(cards []int, guiCount int, feng bool) {
	for i := 0; i < 9; i++ {
		if cards[i] == 0 {
			continue
		}

		cards[i]--
		if !t.tryAdd(cards, guiCount, feng) {
			cards[i]++
			continue
		}
		if guiCount < 8 {
			t.genGui(cards, guiCount+1, feng)
		}
		cards[i]++
	}
}

func (t *Table) tryAdd(cards []int, guiCount int, feng bool) bool {
	for i := 0; i < 9; i++ {
		if cards[i] < 0 || cards[i] > 4 {
			return false
		}
	}
	key := t.genKey(cards)
	if feng {
		if t.keyFengGuiDic[guiCount] == nil {
			t.keyFengGuiDic[guiCount] = make(map[string]bool)
		}
		_, ok := t.keyFengGuiDic[guiCount][key]
		if ok {
			return false
		}
		t.keyFengGuiDic[guiCount][key] = true
	}
	if t.keyGuiDic[guiCount] == nil {
		t.keyGuiDic[guiCount] = make(map[string]bool)
	}
	_, ok := t.keyGuiDic[guiCount][key]
	if ok {
		return false
	}
	t.keyGuiDic[guiCount][key] = true
	return true
}
