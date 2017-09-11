package vt100

import (
	"errors"
	//	"fmt"
	"math/rand"
	"testing"
	"time"
)

func transact(seq string) (*Token, error) {
	vtlx := NewLexer()
	defer vtlx.Rundown()

	for _, c := range seq {
		vtlx.SendChar(byte(c))
	}

	select {
	case tok := <-vtlx.Output:
		return tok, nil

	case <-time.After(100 * time.Millisecond):
		return nil, errors.New("timeout")
	}
}

func testCase(t *testing.T, seq string, val TokVal) {
	tok, err := transact(seq)
	if err != nil {
		t.Errorf("error: %s\n", err)
		t.Errorf("error: expected %s, got nothing\nseq=%#v\n",
			val, seq)
	} else if tok == nil {
		t.Errorf("nil token pointer")
	} else {
		if tok.Value != val {
			t.Errorf("error: expected %s, got %s\nseq=%#v\n",
				val, tok.Value, seq)
		}
	}
}

func TestVT(t *testing.T) {
	testCase(t, "A", 'A')
	testCase(t, "\033[H", CursorHome)
	testCase(t, "\033[20h", SetNL)
	testCase(t, "\033[?1h", SetAppl)
	testCase(t, "\033[?3h", SetCol)
	testCase(t, "\033[?4h", SetSmooth)
	testCase(t, "\033[?5h", SetRevScrn)
	testCase(t, "\033[?6h", SetOrgRel)
	testCase(t, "\033[?7h", SetWrap)
	testCase(t, "\033[?8h", SetRep)
	testCase(t, "\033[?9h", SetInter)

	testCase(t, "\033[20l", SetLF)
	testCase(t, "\033[?1l", SetCursor)
	testCase(t, "\033[?2l", SetVT52)
	testCase(t, "\033[?3l", ResetCol)
	testCase(t, "\033[?4l", SetJump)
	testCase(t, "\033[?5l", SetNormScrn)
	testCase(t, "\033[?6l", SetOrgAbs)
	testCase(t, "\033[?7l", ResetWrap)
	testCase(t, "\033[?8l", ResetRep)
	testCase(t, "\033[?9l", ResetInter)

	testCase(t, "\033=", AltKeypad)
	testCase(t, "\033>", NumKeypad)

	testCase(t, "\033(A", SetUKG0)
	testCase(t, "\033)A", SetUKG1)
	testCase(t, "\033(B", SetUSG0)
	testCase(t, "\033)B", SetUSG1)
	testCase(t, "\033(0", SetSpecG0)
	testCase(t, "\033)0", SetSpecG1)
	testCase(t, "\033(1", SetAltG0)
	testCase(t, "\033)1", SetAltG1)
	testCase(t, "\033(2", SetAltSpecG0)
	testCase(t, "\033)2", SetAltSpecG1)

	testCase(t, "\033N", SetSS2)
	testCase(t, "\033O", SetSS3)

	testCase(t, "\033[m", ModesOff)
	testCase(t, "\033[0m", ModesOff)
	testCase(t, "\033[1m", Bold)
	testCase(t, "\033[2m", LowInt)
	testCase(t, "\033[4m", Underline)
	testCase(t, "\033[5m", Blink)
	testCase(t, "\033[7m", Reverse)
	testCase(t, "\033[8m", Invisible)

	testCase(t, "\033[13;17r", SetWin)
	testCase(t, "\033[3A", CursorUp)
	testCase(t, "\033[4B", CursorDn)
	testCase(t, "\033[5C", CursorRt)
	testCase(t, "\033[6D", CursorLf)
	testCase(t, "\033[H", CursorHome)
	testCase(t, "\033[;H", CursorHome)
	testCase(t, "\033[13;17H", CursorPos)
	testCase(t, "\033[f", HvHome)
	testCase(t, "\033[;f", HvHome)
	testCase(t, "\033[13;17f", HvPos)
	testCase(t, "\033D", Index)
	testCase(t, "\033M", RevIndex)
	testCase(t, "\033E", NextLine)
	testCase(t, "\0337", SaveCursor)
	testCase(t, "\0338", RestoreCursor)

	testCase(t, "\033H", TabSet)
	testCase(t, "\033[g", TabClr)
	testCase(t, "\033[0g", TabClr)
	testCase(t, "\033[3g", TabClrAll)

	testCase(t, "\033#3", DhTop)
	testCase(t, "\033#4", DhBot)
	testCase(t, "\033#5", Swsh)
	testCase(t, "\033#6", Dwsh)

	testCase(t, "\033[K", ClearEOL)
	testCase(t, "\033[0K", ClearEOL)
	testCase(t, "\033[1K", ClearBOL)
	testCase(t, "\033[2K", ClearLine)

	testCase(t, "\033[J", ClearEOS)
	testCase(t, "\033[0J", ClearEOS)
	testCase(t, "\033[1J", ClearBOS)
	testCase(t, "\033[2J", ClearScreen)

	testCase(t, "\0335n", DevStat)
	testCase(t, "\0336n", GetCursor)

	testCase(t, "\033[c", Ident)
	testCase(t, "\033[0c", Ident)

	testCase(t, "\033c", Reset)

	testCase(t, "\033#8", Align)
	testCase(t, "\033[2;1y", TestPU)
	testCase(t, "\033[2;2y", TestLB)
	testCase(t, "\033[2;9y", TestPURep)
	testCase(t, "\033[2;10y", TestLBRep)

	testCase(t, "\033[0q", LedsOff)
	testCase(t, "\033[1q", Led1)
	testCase(t, "\033[2q", Led2)
	testCase(t, "\033[3q", Led3)
	testCase(t, "\033[4q", Led4)

	// A totally random test to ensure that the
	// lexer does not lock up during character
	// injection.
	vtlx := NewLexer()
	now := time.Now()
	rand.Seed(int64(now.Nanosecond()))
	for i := 0; i < 1000000; i++ {
		// every so often, inject a esc-[ sequence
		if (i & 0x4) != 0 {
			vtlx.SendChar(0x1b)
			vtlx.SendChar('[')
		}

		// generate, send random char value
		c := byte(rand.Int())
		vtlx.SendChar(c)

		// discard tokens coming back asynchronously
		select {
		case <-vtlx.Output:
		default:
		}
	}

	for { // catch any straglers
		select {
		case <-vtlx.Output:
		case <-time.After(time.Millisecond):
			goto done
		}
	}
done:
	vtlx.Rundown()
}
