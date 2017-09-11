package vt100

import (
	"bytes"
	"fmt"
	"sync"
	"unicode"
)

// TokVal contains a unique integer value for all vt100 escape
// sequences listed below.
type TokVal int

// Lexical token values for all vt100 escape sequences
// (const pool)
const (
	Align TokVal = -(iota + 1) // starts at -1 and goes to -n
	AltKeypad
	Blink
	Bold
	ClearBOL
	ClearBOS
	ClearEOL
	ClearEOS
	ClearLine
	ClearScreen
	CursorDn
	CursorHome
	CursorLf
	CursorPos
	CursorRt
	CursorUp
	DevStat
	DhBot
	DhTop
	Dwsh
	GetCursor
	HvHome
	HvPos
	Ident
	Index
	Invisible
	Led1
	Led2
	Led3
	Led4
	LedsOff
	LowInt
	ModesOff
	NextLine
	NumKeypad
	Reset
	ResetCol
	ResetInter
	ResetRep
	ResetWrap
	RestoreCursor
	Reverse
	RevIndex
	SaveCursor
	SetAltG0
	SetAltG1
	SetAltSpecG0
	SetAltSpecG1
	SetAppl
	SetCol
	SetCursor
	SetInter
	SetJump
	SetLF
	SetNL
	SetNormScrn
	SetOrgAbs
	SetOrgRel
	SetRep
	SetRevScrn
	SetSmooth
	SetSpecG0
	SetSpecG1
	SetSS2
	SetSS3
	SetUKG0
	SetUKG1
	SetUSG0
	SetUSG1
	SetVT52
	SetWin
	SetWrap
	Swsh
	TabClr
	TabClrAll
	TabSet
	TestLB
	TestLBRep
	TestPU
	TestPURep
	Underline
)

// Note that this list has to directly correspond to the above
// list of constants in both order and number.
// (literal pool)
var data = bytes.NewBufferString(`Align AltKeypad Blink Bold ClearBOL
ClearBOS ClearEOL ClearEOS ClearLine ClearScreen CursorDn CursorHome
CursorLf CursorPos CursorRt CursorUp DevStat DhBot DhTop Dwsh GetCursor
HvHome HvPos Ident Index Invisible Led1 Led2 Led3 Led4 LedsOff LowInt
ModesOff NextLine NumKeypad Reset ResetCol ResetInter ResetRep ResetWrap
RestoreCursor Reverse RevIndex SaveCursor SetAltG0 SetAltG1 SetAltSpecG0
SetAltSpecG1 SetAppl SetCol SetCursor SetInter SetJump SetLF SetNL
SetNormScrn SetOrgAbs SetOrgRel SetRep SetRevScrn SetSmooth SetSpecG0
SetSpecG1 SetSS2 SetSS3 SetUKG0 SetUKG1 SetUSG0 SetUSG1 SetVT52 SetWin
SetWrap Swsh TabClr TabClrAll TabSet TestLB TestLBRep TestPU TestPURep
Underline`)

var labelMap = make(map[TokVal]string)

// To support the String() methods, create a map of token values to
// their literal counterparts.
func init() {
	var t TokVal = -1
	for {
		var s string
		n, err := fmt.Fscan(data, &s)
		if (err == nil) && (n == 1) {
			labelMap[t] = s
			t--
		} else {
			break
		}
	}

	if len(labelMap) != 81 {
		panic("Symbol count changed; verify const pool w/literal pool")
	}
}

func (t TokVal) String() string {
	l, ok := labelMap[t]
	if !ok {
		return "?"
	}
	return l
}

// Token encapsulates all salient aspects regarding a received
// VT-100 escape sequence
type Token struct {
	Value  TokVal // unique integer value
	Params []int  // Parameters, if any (i.e. cursor positioning, etc.)
	seq    []byte // captured escape sequence
}

func (t Token) String() string {
	return fmt.Sprintf("%s Params: %v, Byte Seq: %v", t.Value, t.Params, t.seq)
}

// Lexer holds the state information for our VT-100 lexer
type Lexer struct {
	input           chan byte
	Output          chan *Token
	paramCharsAccum []byte
	params          []int
	seq             []byte
	rundown         chan struct{}
	wg              sync.WaitGroup
}

// NewLexer creates a new VT-100 lexer state machine
func NewLexer() *Lexer {
	l := new(Lexer)
	l.input = make(chan byte, 10)
	l.Output = make(chan *Token, 10)
	l.rundown = make(chan struct{})
	l.wg.Add(1)
	go l.run()
	return l
}

// SendChar sends a character to the lexer
func (l *Lexer) SendChar(c byte) {
	l.input <- c
}

// GetToken waits (i.e. blocks) for the next token to be generated
// by the lexer.  This may not always be desirable.
// NOTE: In the Lexer struct, the Output channel is made public so
// other packages may access this channel from within a select statement.
func (l *Lexer) GetToken() *Token {
	return <-l.Output
}

// Rundown tears down the lexer
func (l *Lexer) Rundown() {
	l.rundown <- struct{}{}
	l.wg.Wait()
	close(l.input)
	close(l.Output)
	close(l.rundown)
}

// stateFn represents the state of the lexical scanner
// as a function that returns the next state.
type stateFn func(c byte) stateFn

// run lexes the input by executing state functions until
// the state is nil.
func (l *Lexer) run() {
	defer l.wg.Done()

	for state := l.ground; state != nil; {
		select {
		case c := <-l.input:
			c &= 0x7f
			if l.seq != nil {
				l.seq = append(l.seq, c)
			}
			state = state(c)

		case <-l.rundown:
			return
		}
	}
}

func (l *Lexer) send(tv TokVal) {
	l.Output <- &Token{tv, l.params, l.seq}
}

// Ground state of the Lexer
func (l *Lexer) ground(c byte) stateFn {
	l.paramCharsAccum, l.params, l.seq = nil, nil, nil
	if c == 0x1b {
		l.seq = []byte{c}
		return l.intermediateChar
	}
	l.send(TokVal(c))
	return l.ground
}

func (l *Lexer) intermediateChar(c byte) stateFn {
	switch c {
	case '[':
		return l.afterLeftSquareBracket
	case '(':
		return l.afterLeftParen
	case ')':
		return l.afterRightParen
	case '#':
		return l.escPound
	case 'D':
		l.send(Index)
	case 'M':
		l.send(RevIndex)
	case 'N':
		l.send(SetSS2)
	case 'O':
		l.send(SetSS3)
	case 'E':
		l.send(NextLine)
	case '7':
		l.send(SaveCursor)
	case '8':
		l.send(RestoreCursor)
	case '=':
		l.send(AltKeypad)
	case '>':
		l.send(NumKeypad)
	case 'H':
		l.send(TabSet)
	case 'c':
		l.send(Reset)

	default:
		if unicode.IsDigit(rune(c)) {
			return l.escapeDigit // no left-square-bracket
		}
	}
	return l.ground
}

func (l *Lexer) afterLeftSquareBracket(c byte) stateFn {
	switch {
	case unicode.IsLetter(rune(c)): // terminating char
		return l.interpTerm(c)

	case unicode.IsPrint(rune(c)) && !unicode.IsSpace(rune(c)):
		// All other non-whitespace printables
		// Regrettably, IsPrint accepts space chars
		// Simply let chars accumulate within l.seq byte slice
		return l.afterLeftSquareBracket

	default: // discard weird char and reset to ground state
		return l.ground
	}
}

// interpret terminator Letter character
func (l *Lexer) interpTerm(c byte) stateFn {
	// analyze everything after the esc-[ sequence
	body := string(l.seq[2:])

	switch c {
	case 'h':
		switch body {
		case "20h":
			l.send(SetNL)
		case "?1h":
			l.send(SetAppl)
		case "?3h":
			l.send(SetCol)
		case "?4h":
			l.send(SetSmooth)
		case "?5h":
			l.send(SetRevScrn)
		case "?6h":
			l.send(SetOrgRel)
		case "?7h":
			l.send(SetWrap)
		case "?8h":
			l.send(SetRep)
		case "?9h":
			l.send(SetInter)
		}

	case 'l':
		switch body {
		case "20l":
			l.send(SetLF)
		case "?1l":
			l.send(SetCursor)
		case "?2l":
			l.send(SetVT52)
		case "?3l":
			l.send(ResetCol)
		case "?4l":
			l.send(SetJump)
		case "?5l":
			l.send(SetNormScrn)
		case "?6l":
			l.send(SetOrgAbs)
		case "?7l":
			l.send(ResetWrap)
		case "?8l":
			l.send(ResetRep)
		case "?9l":
			l.send(ResetInter)
		}

	case 'm':
		switch body {
		case "m":
			l.send(ModesOff)
		case "0m":
			l.send(ModesOff)
		case "1m":
			l.send(Bold)
		case "2m":
			l.send(LowInt)
		case "4m":
			l.send(Underline)
		case "5m":
			l.send(Blink)
		case "7m":
			l.send(Reverse)
		case "8m":
			l.send(Invisible)
		}

	case 'r':
		var top, bottom byte
		n, err := fmt.Sscanf(body, "%d;%d", &top, &bottom)
		if (err == nil) && (n == 2) { // success case
			l.params = []int{int(top), int(bottom)}
			l.send(SetWin)
		}

	case 'A':
		var lines byte
		n, err := fmt.Sscanf(body, "%d", &lines)
		if (err == nil) && (n == 1) { // success case
			l.params = []int{int(lines)}
			l.send(CursorUp)
		}

	case 'B':
		var lines byte
		n, err := fmt.Sscanf(body, "%d", &lines)
		if (err == nil) && (n == 1) { // success case
			l.params = []int{int(lines)}
			l.send(CursorDn)
		}

	case 'C':
		var cols byte
		n, err := fmt.Sscanf(body, "%d", &cols)
		if (err == nil) && (n == 1) { // success case
			l.params = []int{int(cols)}
			l.send(CursorRt)
		}

	case 'D':
		var cols byte
		n, err := fmt.Sscanf(body, "%d", &cols)
		if (err == nil) && (n == 1) { // success case
			l.params = []int{int(cols)}
			l.send(CursorLf)
		}

	case 'H':
		if (body == "H") || (body == ";H") {
			l.send(CursorHome)
		} else {
			var v, h byte
			n, err := fmt.Sscanf(body, "%d;%d", &v, &h)
			if (err == nil) && (n == 2) {
				l.params = []int{int(v), int(h)}
				l.send(CursorPos)
			}
		}

	case 'f':
		if (body == "f") || (body == ";f") {
			l.send(HvHome)
		} else {
			var v, h byte
			n, err := fmt.Sscanf(body, "%d;%d", &v, &h)
			if (err == nil) && (n == 2) {
				l.params = []int{int(v), int(h)}
				l.send(HvPos)
			}
		}

	case 'g':
		switch body {
		case "g":
			l.send(TabClr)

		case "0g":
			l.send(TabClr)

		case "3g":
			l.send(TabClrAll)
		}

	case 'K':
		switch body {
		case "K":
			l.send(ClearEOL)

		case "0K":
			l.send(ClearEOL)

		case "1K":
			l.send(ClearBOL)

		case "2K":
			l.send(ClearLine)
		}

	case 'J':
		switch body {
		case "J":
			l.send(ClearEOS)

		case "0J":
			l.send(ClearEOS)

		case "1J":
			l.send(ClearBOS)

		case "2J":
			l.send(ClearScreen)
		}

	case 'c':
		switch body {
		case "c":
			l.send(Ident)

		case "0c":
			l.send(Ident)
		}

	case 'y':
		switch body {
		case "2;1y":
			l.send(TestPU)

		case "2;2y":
			l.send(TestLB)

		case "2;9y":
			l.send(TestPURep)

		case "2;10y":
			l.send(TestLBRep)
		}

	case 'q':
		switch body {
		case "0q":
			l.send(LedsOff)

		case "1q":
			l.send(Led1)

		case "2q":
			l.send(Led2)

		case "3q":
			l.send(Led3)

		case "4q":
			l.send(Led4)
		}
	}

	return l.ground
}

func (l *Lexer) escPound(c byte) stateFn {
	switch c {
	case '3':
		l.send(DhTop)
	case '4':
		l.send(DhBot)
	case '5':
		l.send(Swsh)
	case '6':
		l.send(Dwsh)
	case '8':
		l.send(Align)
	}
	return l.ground
}

func (l *Lexer) afterLeftParen(c byte) stateFn {
	switch c {
	case 'A':
		l.send(SetUKG0)
	case 'B':
		l.send(SetUSG0)
	case '0':
		l.send(SetSpecG0)
	case '1':
		l.send(SetAltG0)
	case '2':
		l.send(SetAltSpecG0)
	}
	return l.ground
}

func (l *Lexer) afterRightParen(c byte) stateFn {
	switch c {
	case 'A':
		l.send(SetUKG1)
	case 'B':
		l.send(SetUSG1)
	case '0':
		l.send(SetSpecG1)
	case '1':
		l.send(SetAltG1)
	case '2':
		l.send(SetAltSpecG1)
	}
	return l.ground
}

func (l *Lexer) escapeDigit(c byte) stateFn {
	body := string(l.seq[1:])
	switch body {
	case "5n":
		l.send(DevStat)

	case "6n":
		l.send(GetCursor)
	}
	return l.ground
}
