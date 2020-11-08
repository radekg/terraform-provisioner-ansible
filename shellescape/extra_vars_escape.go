package shellescape

import "io"

// ShellEscape defines a shell escape interface.
type ShellEscape interface {
	// Safe returns a safe to use copy of the input.
	Safe() string
}

type defaultEscape struct {
	delimitter rune
	input      []rune
	pos        int
	max        int

	seqEscaped        []rune
	seqLookAheadLong  string
	seqLookAheadShort string
}

func (p *defaultEscape) lookup(count int) (string, error) {
	runes := []rune{}
	pos := p.pos
	for {
		if len(runes) == count {
			break
		}
		if pos >= p.max {
			return string(runes), io.EOF
		}
		runes = append(runes, p.input[pos])
		pos = pos + 1
	}
	return string(runes), io.EOF
}

func (p *defaultEscape) skip(n int) {
	p.pos = p.pos + n
}

func (p *defaultEscape) next() (rune, error) {
	if p.pos < p.max {
		r := p.input[p.pos]
		p.pos = p.pos + 1
		return r, nil
	}
	return ' ', io.EOF
}

func (p *defaultEscape) Safe() string {
	newRunes := []rune{}
	for {

		r, e := p.next()
		if e == io.EOF {
			break
		}

		if r == p.delimitter {
			if lookup, _ := p.lookup(3); lookup == p.seqLookAheadLong {
				// this is okay...
				newRunes = append(newRunes, p.seqEscaped...)
				p.skip(3)
				continue
			}
			if lookup, _ := p.lookup(2); lookup == p.seqLookAheadShort {
				// this is okay...
				newRunes = append(newRunes, p.seqEscaped...)
				p.skip(2)
				continue
			}
			newRunes = append(newRunes, p.seqEscaped...)
			continue
		}

		if r == '\\' {
			if lookup, _ := p.lookup(1); lookup == string([]rune{p.delimitter}) {
				newRunes = append(newRunes, p.seqEscaped...)
				p.skip(1)
				continue
			}
		}

		newRunes = append(newRunes, r)
	}
	return string(newRunes)
}

func newDefaultEscape(input string, delimiter rune) ShellEscape {
	runes := []rune(input)
	return &defaultEscape{
		delimitter:        delimiter,
		input:             runes,
		pos:               0,
		max:               len(runes),
		seqEscaped:        []rune{delimiter, '\\', delimiter, delimiter},
		seqLookAheadLong:  string([]rune{'\\', delimiter, delimiter}),
		seqLookAheadShort: string([]rune{'\\', delimiter}),
	}
}

// NewDoubleQuoteEscape returns a new shell escape for use within double quoted string.
func NewDoubleQuoteEscape(input string) ShellEscape {
	return newDefaultEscape(input, '"')
}

// NewSingleQuoteEscape returns a new shell escape for use within single quoted string.
func NewSingleQuoteEscape(input string) ShellEscape {
	return newDefaultEscape(input, '\'')
}
