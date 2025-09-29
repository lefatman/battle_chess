// cmd/battle_chess/main.go (drop-in)
// No preselected abilities/elements. Players must configure both sides before first move.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	// Adjust these imports to your actual module paths if different.
	"battle_chess_poc/internal/game" // your engine: NewEngine(), SetSideConfig(...), etc.
	_ "battle_chess_poc/internal/game/abilities"
	"battle_chess_poc/internal/httpx"  // your HTTP server wrapper exposing Listen(engine) or similar
	"battle_chess_poc/internal/shared" // enums + parsers (Ability, Element, ParseAbility/ParseElement, *Strings)  // <-- from abilities.go/types.go
)

func main() {
	// Flags (env fallbacks). Default: NO PRECONFIG.
	addr := flag.String("addr", getenv("BCHESS_ADDR", ":8080"), "listen address")
	preconfig := flag.Bool("preconfig", getenb("BCHESS_PRECONFIG", false), "apply side configs on startup (default: false)")
	wAbils := flag.String("white-abilities", getenv("BCHESS_WHITE_ABILITIES", ""), "comma-separated abilities for White (used only if -preconfig)")
	bAbils := flag.String("black-abilities", getenv("BCHESS_BLACK_ABILITIES", ""), "comma-separated abilities for Black (used only if -preconfig)")
	wElem := flag.String("white-element", getenv("BCHESS_WHITE_ELEMENT", ""), "element for White (used only if -preconfig)")
	bElem := flag.String("black-element", getenv("BCHESS_BLACK_ELEMENT", ""), "element for Black (used only if -preconfig)")
	flag.Parse()

	eng := game.NewEngine()

	if *preconfig {
		wa, err := parseAbilitiesCSV(*wAbils)
		fatalIf(err, "white abilities")
		ba, err := parseAbilitiesCSV(*bAbils)
		fatalIf(err, "black abilities")

		we, ok := shared.ParseElement(*wElem)
		fatalIfBool(!ok, fmt.Errorf("invalid white element %q; valid: %v", *wElem, shared.ElementStrings()))
		be, ok := shared.ParseElement(*bElem)
		fatalIfBool(!ok, fmt.Errorf("invalid black element %q; valid: %v", *bElem, shared.ElementStrings()))

		if err := eng.SetSideConfig(shared.White, wa, we); err != nil {
			log.Fatalf("config white: %v", err)
		}
		if err := eng.SetSideConfig(shared.Black, ba, be); err != nil {
			log.Fatalf("config black: %v", err)
		}
		log.Printf("Preconfig ON: White[%v,%s] Black[%v,%s]", wa.Strings(), we, ba.Strings(), be)
	} else {
		// No defaults, nothing selected. UI/API must set both sides before first move.
		log.Printf("No preconfig. Both players must select Ability/Element before the match starts.")
	}

	srv := httpx.NewServer(eng)
	log.Printf("HTTP listening on %s", *addr)
	if err := srv.Listen(*addr); err != nil {
		log.Fatal(err)
	}
}

func parseAbilitiesCSV(s string) (shared.AbilityList, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty ability list")
	}
	parts := strings.Split(s, ",")
	out := make(shared.AbilityList, 0, len(parts))
	for _, p := range parts {
		a, ok := shared.ParseAbility(strings.TrimSpace(p))
		if !ok {
			return nil, fmt.Errorf("invalid ability %q; valid: %v", p, shared.AbilityStrings()) // abilities list from your shared pkg. :contentReference[oaicite:2]{index=2}
		}
		out = append(out, a)
	}
	return out, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenb(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "t", "yes", "y", "on":
			return true
		case "0", "false", "f", "no", "n", "off":
			return false
		}
	}
	return def
}

func fatalIf(err error, label string) {
	if err != nil {
		log.Fatalf("%s: %v", label, err)
	}
}

func fatalIfBool(b bool, err error) {
	if b {
		log.Fatal(err)
	}
}
