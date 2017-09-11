# GoVT100Lexer
GoVT100Lexer is a package that can recognize VT-100 escape sequences within a character stream.

This package is based on a video that Rob Pike gave a few years back.  Rob presented a lexical
analyzer based on a very efficient state machine design written in Go.

I have recently been involved with a project to interface to a legacy device that emits vt100 control sequences.
Ostensibly, this legacy device was hooked to a VT-100 terminal or compatible device.  I wanted a more refined
way of dealing with these escape sequences so that I might capture relavent data from the device's menu system
and event reporting screens.

Given data from a serial connection (or other type of stream), the lexer will recognize VT-100 sequences and
return a token structure containing the sequence of bytes and a unique token value.  The token value could be
used within a switch statement inside the user's application to implement terminal-like functionality.
Sequences that contain parameters (i.e. cursor positioning, etc.) are also sent back to the user application
with the token value.  The behavior is similiar to how a UNIX lex program might operate.

Use this as you see fit.  Contributions and feedback are welcome.

-P.




