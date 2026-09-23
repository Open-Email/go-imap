package imapclient

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Copy sends a COPY command.
func (c *Client) Copy(numSet imap.NumSet, mailbox string) *CopyCommand {
	cmd := &CopyCommand{}
	enc := c.beginCommand(uidCmdName("COPY", imapwire.NumSetKind(numSet)), cmd)
	enc.SP().NumSet(numSet).SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// CopyCommand is a COPY command.
type CopyCommand struct {
	commandBase
	data imap.CopyData
}

func (cmd *CopyCommand) Wait() (*imap.CopyData, error) {
	return &cmd.data, cmd.wait()
}

func readRespCodeCopyUID(dec *imapwire.Decoder) (imap.CopyData, error) {
	var uidValidity uint32
	var source, dest string
	isUIDSetChar := func(ch byte) bool { return ch == '*' || imapwire.IsAtomChar(ch) }
	// Atom parsing deliberately accepts "*" here so malformed advisory data
	// can be consumed and ignored without tearing down a successful command.
	if !dec.ExpectNumber(&uidValidity) || !dec.ExpectSP() || !dec.Expect(dec.Func(&source, isUIDSetChar), "COPYUID source") ||
		!dec.ExpectSP() || !dec.Expect(dec.Func(&dest, isUIDSetChar), "COPYUID destination") {
		return imap.CopyData{}, dec.Err()
	}
	mapping, err := imapwire.ParseUIDMapping(source, dest)
	if err != nil || uidValidity == 0 {
		return imap.CopyData{}, nil
	}
	return imap.CopyData{UIDValidity: uidValidity, UIDMapping: mapping}, nil
}
