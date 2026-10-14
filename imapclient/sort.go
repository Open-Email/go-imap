package imapclient

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// SortOptions contains options for the SORT command.
type SortOptions struct {
	// The search criteria for the messages to sort.
	SearchCriteria *imap.SearchCriteria
	// A list of criteria to sort by.
	SortCriteria []imap.SortCriterion
	// Return options for ESORT. If any are set, an extended SORT is used.
	Return imap.SortOptions
}

// SortData is the data returned by a SORT or ESORT command.
type SortData struct {
	// A list of matching message numbers, in sorted order.
	// Populated if Return.All is true (default for SORT).
	// Either SeqNums or UIDs is populated.
	SeqNums []uint32
	UIDs    []imap.UID

	// The following fields are only populated for ESORT.
	Min   uint32
	Max   uint32
	Count uint32
}

func (c *Client) sort(numKind imapwire.NumKind, options *SortOptions) *SortCommand {
	cmd := &SortCommand{numKind: numKind}
	enc := c.beginCommand(uidCmdName("SORT", numKind), cmd)

	isESort := options.Return.ReturnMin || options.Return.ReturnMax || options.Return.ReturnAll || options.Return.ReturnCount
	if isESort {
		enc.SP().Atom("RETURN").SP()
		var returnOpts []string
		if options.Return.ReturnMin {
			returnOpts = append(returnOpts, "MIN")
		}
		if options.Return.ReturnMax {
			returnOpts = append(returnOpts, "MAX")
		}
		if options.Return.ReturnAll {
			returnOpts = append(returnOpts, "ALL")
		}
		if options.Return.ReturnCount {
			returnOpts = append(returnOpts, "COUNT")
		}
		enc.List(len(returnOpts), func(i int) {
			enc.Atom(returnOpts[i])
		})
	}

	enc.SP().List(len(options.SortCriteria), func(i int) {
		criterion := options.SortCriteria[i]
		if criterion.Reverse {
			enc.Atom("REVERSE").SP()
		}
		enc.Atom(string(criterion.Key))
	})
	enc.SP().Atom("UTF-8").SP()
	writeSearchKey(enc.Encoder, options.SearchCriteria, c.Caps().Has(imap.CapCondStore))
	enc.end()
	return cmd
}

func (c *Client) handleSort() error {
	cmd := findPendingCmdByType[*SortCommand](c)
	for c.dec.SP() {
		var num uint32
		if !c.dec.ExpectNumber(&num) {
			return c.dec.Err()
		}
		if cmd != nil {
			if cmd.numKind == imapwire.NumKindSeq {
				cmd.data.SeqNums = append(cmd.data.SeqNums, num)
			} else {
				cmd.data.UIDs = append(cmd.data.UIDs, imap.UID(num))
			}
		}
	}
	return nil
}

// Sort sends a SORT command.
//
// This command requires support for the SORT extension.
func (c *Client) Sort(options *SortOptions) *SortCommand {
	return c.sort(imapwire.NumKindSeq, options)
}

// UIDSort sends a UID SORT command.
//
// See Sort.
func (c *Client) UIDSort(options *SortOptions) *SortCommand {
	return c.sort(imapwire.NumKindUID, options)
}

// SortCommand is a SORT command.
type SortCommand struct {
	commandBase
	numKind imapwire.NumKind
	data    SortData
}

func (cmd *SortCommand) Wait() (*SortData, error) {
	err := cmd.wait()
	return &cmd.data, err
}
