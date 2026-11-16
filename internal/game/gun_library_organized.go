package game

var organizedGunLibraryTabs = []struct {
	label      string
	category   DevelopedWeaponCategory
	throwables bool
}{
	{"ALL", 0, false},
	{"PISTOLS", DevelopedCategoryPistol, false},
	{"SMG", DevelopedCategorySMG, false},
	{"RIFLES", DevelopedCategoryRifle, false},
	{"LMG", DevelopedCategoryLMG, false},
	{"SNIPERS", DevelopedCategorySniper, false},
	{"SHOTGUNS", DevelopedCategoryShotgun, false},
	{"LAUNCHERS", DevelopedCategoryLauncher, false},
	{"SPECIAL", DevelopedCategorySpecial, false},
	{"THROWABLES", 0, true},
}

const organizedGunLibraryPageSize = 20

func sourceWeaponCategory(number int) DevelopedWeaponCategory {
	switch {
	case number >= 1 && number <= 6:
		return DevelopedCategoryPistol
	case number == 12 || number == 19 || (number >= 21 && number <= 32) || (number >= 67 && number <= 71):
		return DevelopedCategorySMG
	case number == 55 || number == 57 || (number >= 64 && number <= 66):
		return DevelopedCategoryLMG
	case number == 11 || number == 14 || number == 16 || (number >= 33 && number <= 43) || (number >= 77 && number <= 81):
		return DevelopedCategorySniper
	case number == 13 || number == 15 || number == 18 || (number >= 44 && number <= 54) || (number >= 82 && number <= 86):
		return DevelopedCategoryShotgun
	default:
		return DevelopedCategoryRifle
	}
}

func organizedGunLibraryNumbers(category DevelopedWeaponCategory) []int {
	out := make([]int, 0, 186)
	appendIf := func(number int, weaponCategory DevelopedWeaponCategory) {
		if category == 0 || category == weaponCategory {
			out = append(out, number)
		}
	}
	for number := 1; number <= 6; number++ {
		appendIf(number, DevelopedCategoryPistol)
	}
	for number := 10; number <= 86; number++ {
		appendIf(number, sourceWeaponCategory(number))
	}
	for _, number := range allDevelopedWeaponNumbers() {
		spec := developedWeaponSpecs[number]
		category := spec.Category
		if category == DevelopedCategoryMelee {
			category = DevelopedCategorySpecial // the bat sits with the specials
		}
		appendIf(number, category)
	}
	return out
}

func organizedGunLibraryPageCount(category DevelopedWeaponCategory) int {
	count := len(organizedGunLibraryNumbers(category))
	if count == 0 {
		return 1
	}
	return (count + organizedGunLibraryPageSize - 1) / organizedGunLibraryPageSize
}

func (g *Game) organizedGunLibraryThrowableTab() bool {
	return g.gunLibraryCategory >= 0 && g.gunLibraryCategory < len(organizedGunLibraryTabs) &&
		organizedGunLibraryTabs[g.gunLibraryCategory].throwables
}

func (g *Game) changeGunLibraryPage(delta int) {
	if g.organizedGunLibraryThrowableTab() {
		g.gunLibraryPage = 0
		return
	}
	category := DevelopedWeaponCategory(0)
	if g.gunLibraryCategory >= 0 && g.gunLibraryCategory < len(organizedGunLibraryTabs) {
		category = organizedGunLibraryTabs[g.gunLibraryCategory].category
	}
	count := organizedGunLibraryPageCount(category)
	if count <= 1 {
		g.gunLibraryPage = 0
		return
	}
	g.gunLibraryPage = (g.gunLibraryPage + delta) % count
	if g.gunLibraryPage < 0 {
		g.gunLibraryPage += count
	}
}
