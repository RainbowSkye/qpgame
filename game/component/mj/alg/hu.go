package alg

import (
	"game/component/mj/mp"
)

var table = NewTable()

type HuLogic struct {
}

func NewHuLogic() *HuLogic {
	return &HuLogic{}
}

func (h *HuLogic) CheckHu(cards []mp.CardID, guiList []mp.CardID, card mp.CardID) bool {
	if card > 0 && card < 36 && len(cards) < 14 {
		cards = append(cards, card)
	}
	return h.isHu(cards, guiList)
}

func (h *HuLogic) isHu(cardList []mp.CardID, list []mp.CardID) bool {
	cards := [][]int{
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	guiCount := 0
	for _, v := range cardList {
		if v == mp.Zhong {
			guiCount++
		} else {
			i := v / 10
			j := v%10 - 1
			cards[i][j]++
		}
	}

	cardData := &CardData{
		guiCount: guiCount,
		jiang:    false,
	}

	for i := 0; i < 4; i++ {
		feng := i == 3
		cardData.cards = cards[i]
		if !h.checkCards(cardData, 0, feng) {
			return false
		}
	}
	if !cardData.jiang && cardData.guiCount%3 == 2 {
		return true
	}
	if cardData.jiang && cardData.guiCount%3 == 0 {
		return true
	}

	return false
}

func (h *HuLogic) checkCards(data *CardData, guiCount int, feng bool) bool {
	totalCardCount := table.calcTotalCardCount(data.cards)
	if totalCardCount == 0 {
		return true
	}
	// 查表 如果表没有 那么就加一个鬼
	if !table.findCards(data.cards, guiCount, feng) {
		if guiCount < data.guiCount {
			// 递归 每次鬼+1
			return h.checkCards(data, guiCount+1, feng)
		} else {
			return false
		}
	} else {
		// 将只能有一个
		if (totalCardCount+guiCount)%3 == 2 {
			if !data.jiang {
				data.jiang = true
			} else if guiCount < data.guiCount {
				// 需要再次尝试+1鬼 看是否胡 只能有一个将
				return h.checkCards(data, guiCount+1, feng)
			} else {
				return false
			}
		}
		data.guiCount = data.guiCount - guiCount
	}
	return true
}

type CardData struct {
	cards    []int
	guiCount int
	jiang    bool
}
