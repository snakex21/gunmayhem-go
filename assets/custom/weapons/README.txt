GUN MAYHEM - CUSTOM WEAPONS 87+

Nowe bronie 87..185 maja definicje, mechaniki i wygenerowane sprite'y.
Sa wlaczane do normalnej puli gry, trybow online i odpowiednich modyfikatorow.

Jak wlaczyc konkretna bron:
1. Odszukaj jej AssetDir w internal/game/developed_weapons.go.
2. Utworz katalog:
   assets/custom/weapons/<AssetDir>/
3. Dodaj przynajmniej:
   1.png

Przyklad RPG-7:
   assets/custom/weapons/rpg7/1.png

Po wykryciu 1.png bron moze wejsc do developedCrateWeapons().
Kolejne klatki animacji mozna dodac jako 2.png, 3.png itd. Jesli ich nie ma, renderer uzywa 1.png jako fallback.

Status (aktualny):
- GM1 1..66: oryginal
- GM2 67..86: oryginal GM2
- CUSTOM 87..185: 99 aktywnych broni z osobnymi sprite'ami PNG

Rakiety maja animowany plomien i dym podczas lotu. Pociski granatnikow
obracaja sie zgodnie z trajektoria. Kusze, flary i bron energetyczna maja
odrebne, widoczne pociski.

Lista i parametry nowych broni:
   internal/game/developed_weapons.go

Wazne:
- Nie wrzucaj nowych assetow do assets/gm2 ani do katalogu GM1.
- CUSTOM jest osobnym namespace, zeby nie naruszyc zgodnosci 1:1 z oryginalem.
