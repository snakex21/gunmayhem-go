package game

import "testing"

func TestEverySourceWeaponHasDevelopedDisplayName(t *testing.T) {
	for number, def := range weaponCatalog {
		name := def.DisplayName()
		if name == "" || name == def.Name {
			if number == 9 { // De-Materializer is fictional and keeps its source name.
				continue
			}
			t.Errorf("weapon %d (%s) has no distinct developed display name", number, def.Name)
		}
	}
}

func TestKnownRealWeaponNames(t *testing.T) {
	checks := map[int]string{
		1:  "Colt M1911A1",
		4:  "Taurus Raging Bull M444",
		10: "AK-47",
		20: "Bushmaster M17S",
		30: "KRISS Vector",
		45: "Kel-Tec KSG-12",
		62: "FN F2000",
		66: "Steyr AUG HBAR",
	}
	for number, want := range checks {
		def, ok := WeaponByNumber(number)
		if !ok {
			t.Fatalf("weapon %d missing", number)
		}
		if got := def.DisplayName(); got != want {
			t.Errorf("weapon %d display name=%q want %q", number, got, want)
		}
	}
}
