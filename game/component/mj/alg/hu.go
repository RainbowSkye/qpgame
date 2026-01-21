package alg

import "game/component/mj/mp"

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

func (h *HuLogic) isHu(cards []mp.CardID, list []mp.CardID) bool {
	return false
}
