package sz

import (
	"math/rand/v2"
	"sort"
	"sync"
)

type Logic struct {
	sync.RWMutex
	cards []int // 52张牌
}

func NewLogic() *Logic {
	return &Logic{
		cards: make([]int, 0),
	}
}

// washCards  方块 梅花 红桃 黑桃
func (l *Logic) washCards() {
	l.cards = []int{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d,
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d,
	}
	for i, v := range l.cards {
		random := rand.IntN(len(l.cards))
		l.cards[i] = l.cards[random]
		l.cards[random] = v
	}
}

// getCards 获取三张手牌
func (l *Logic) getCards() []int {
	cards := make([]int, 3)
	l.RLock()
	defer l.RUnlock()
	for i := 0; i < 3; i++ {
		if len(cards) == 0 {
			break
		}
		card := l.cards[len(l.cards)-1]
		l.cards = l.cards[:len(l.cards)-1]
		cards[i] = card
	}
	return cards
}

// CompareCards 比牌操作  0 - 平局， 1 - 当前用户赢， -1 - 当前用户输
func (l *Logic) CompareCards(cards1 []int, cards2 []int) int {
	// 判断牌型
	type1 := l.getCardsType(cards1)
	type2 := l.getCardsType(cards2)
	if type1 != type2 {
		return int(type1 - type2)
	}

	// 牌型相同，比较牌面值
	// 先比较是不是对子
	value1 := l.getCardsValue(cards1)
	value2 := l.getCardsValue(cards2)
	if type1 == DuiZi {
		duiZi1, danZhang1 := l.getDuiZi(value1)
		duiZi2, danZhang2 := l.getDuiZi(value2)
		if duiZi1 == duiZi2 {
			return danZhang1 - danZhang2
		}
		if duiZi1 == 0x01 {
			duiZi1 = 0x0f
		}
		if duiZi2 == 0x01 {
			duiZi2 = 0x0f
		}
		return duiZi1 - duiZi2
	}
	// 比较最大的牌面
	if value1[1] == 0x01 {
		value1[1] = 0x0f
	}
	if value1[2] == 0x01 {
		value1[2] = 0x0f
	}
	return value1[2] - value2[2]
}

func (l *Logic) getCardsType(cards []int) CardsType {
	// 获取牌面数字
	num1 := l.getCardNumber(cards[0])
	num2 := l.getCardNumber(cards[1])
	num3 := l.getCardNumber(cards[2])

	// 判断是不是豹子
	if num1 == num2 && num2 == num3 {
		return BaoZi
	}

	// 获取花色
	color1 := l.getCardColor(cards[0])
	color2 := l.getCardColor(cards[1])
	color3 := l.getCardColor(cards[2])
	isJinHua := false
	// 判断是不是金花
	if color1 == color2 && color2 == color3 {
		isJinHua = true
	}
	isShunZi := false
	// 判断是不是顺子	特殊情况 - QKA
	cardsNum := []int{num1, num2, num3}
	sort.Ints(cardsNum)
	if cardsNum[0]+1 == cardsNum[1] && cardsNum[1]+1 == cardsNum[2] {
		isShunZi = true
	} else if cardsNum[0] == 0x01 && cardsNum[1] == 0x0c && cardsNum[1] == 0x0d {
		isShunZi = true
	}
	if isJinHua && isShunZi {
		return ShunJin
	}
	if isShunZi {
		return ShunZi
	}

	// 判断是不是对子
	if cardsNum[0] == cardsNum[1] || cardsNum[1] == cardsNum[2] {
		return DuiZi
	}

	return DanZhang
}

func (l *Logic) getCardNumber(card int) int {
	return card & 0x0F
}

func (l *Logic) getCardColor(card int) int {
	return card & 0xF0
}

func (l *Logic) getCardsValue(cards []int) []int {
	num1 := l.getCardNumber(cards[0])
	num2 := l.getCardNumber(cards[1])
	num3 := l.getCardNumber(cards[2])
	cardsNum := []int{num1, num2, num3}
	sort.Ints(cardsNum)
	return cardsNum
}

// 对于已经排序的牌 获取对子
func (l *Logic) getDuiZi(cards []int) (int, int) {
	// AAB
	if cards[0] == cards[1] {
		return cards[0], cards[2]
	}
	// BAA
	return cards[1], cards[0]
}
