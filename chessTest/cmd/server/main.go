// path: chessTest/cmd/server/main.go
// cmd/battle_chess/main.go (drop-in)
// No preselected abilities/elements. Players must configure both sides before first move.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"battle_chess_poc/internal/game"
	"battle_chess_poc/internal/httpx"
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

		we, ok := game.ParseElement(*wElem)
		fatalIfBool(!ok, fmt.Errorf("invalid white element %q; valid: %v", *wElem, game.ElementStrings()))
		be, ok := game.ParseElement(*bElem)
		fatalIfBool(!ok, fmt.Errorf("invalid black element %q; valid: %v", *bElem, game.ElementStrings()))

		if err := eng.SetSideConfig(game.White, wa, we); err != nil {
			log.Fatalf("config white: %v", err)
		}
		if err := eng.SetSideConfig(game.Black, ba, be); err != nil {
			log.Fatalf("config black: %v", err)
		}
		log.Printf("Preconfig ON: White[%v,%s] Black[%v,%s]", wa.Strings(), we, ba.Strings(), be)
	} else {
		// No defaults, nothing selected. UI/API must set both sides before first move.
		log.Printf("No preconfig. Both players must select Ability/Element before the match starts.")
	}

	srv, err := httpx.NewServer(eng)
	if err != nil {
		log.Fatalf("http init: %v", err)
	}
	log.Printf("HTTP listening on %s", *addr)
	if err := srv.Listen(*addr); err != nil {
		log.Fatal(err)
	}
}

func parseAbilitiesCSV(s string) (game.AbilityList, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty ability list")
	}
	parts := strings.Split(s, ",")
	out := make(game.AbilityList, 0, len(parts))
	for _, p := range parts {
		a, ok := game.ParseAbility(strings.TrimSpace(p))
		if !ok {
			return nil, fmt.Errorf("invalid ability %q; valid: %v", p, game.AbilityStrings())
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
