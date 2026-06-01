package chat

import (
	"fmt"

	"github.com/NuuttiSir/tuiwatchers/internal/emotes"
)

func RenderLine(line ChatLine) {
	fmt.Printf("\x1b[1m%s\x1b[0m: ", line.User) // bold username
	for _, part := range line.Parts {
		switch part.Kind {
		case "text":
			fmt.Print(part.Text)
		case "emote":
			if img, ok := emotes.EmoteImage(part.EmoteID); ok {
				fmt.Print(emotes.KittyInlineImage(img), " ")
			} else {
				fmt.Printf("[%s]", part.Text) // fallback until cached
			}
		}
	}
}
