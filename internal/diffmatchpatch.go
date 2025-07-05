package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type DMPOp int

const (
	DMPOpEqual DMPOp = iota
	DMPOpDelete
	DMPOpInsert
)

func (p DMPOp) String() string {
	switch p {
	case DMPOpDelete:
		return "-"
	case DMPOpInsert:
		return "+"
	default:
		return " "
	}
}

func (p DMPOp) Color() func(string) string {
	switch p {
	case DMPOpDelete:
		return redString
	case DMPOpInsert:
		return greenString
	default:
		return identString
	}
}

type DMPHunk struct {
	Op   DMPOp
	Body string
}

func (h *DMPHunk) clone() *DMPHunk {
	return &DMPHunk{
		Op:   h.Op,
		Body: h.Body,
	}
}

func (h *DMPHunk) IntoString(color bool) string {
	ss := strings.Split(h.Body, "\n")
	xs := make([]string, len(ss))
	for i, s := range ss {
		if i == len(ss)-1 && s == "" {
			xs[i] = s
			continue
		}
		x := h.Op.String() + s
		if color {
			x = h.Op.Color()(x)
		}
		xs[i] = x
	}
	return strings.Join(xs, "\n")
}

func (h *DMPHunk) countLines() int {
	if h.Body == "" {
		return 0
	}
	n := strings.Count(h.Body, "\n")
	if n == 0 {
		return 1
	}
	return n
}

func (h *DMPHunk) head(n int) string { return HeadString(h.Body, "\n", n) }
func (h *DMPHunk) tail(n int) string { return TailString(h.Body, "\n", n) }

func (h *DMPHunk) debugString() string {
	n := strconv.Itoa(h.countLines())
	switch h.Op {
	case DMPOpDelete:
		return "D" + n
	case DMPOpInsert:
		return "I" + n
	default:
		return "E" + n
	}
}

type DMPPatch struct {
	LeftStart   int
	RightStart  int
	LeftLength  int
	RightLength int
	Hunks       []*DMPHunk
}

func (p *DMPPatch) clone() *DMPPatch {
	hunks := make([]*DMPHunk, len(p.Hunks))
	for i, h := range p.Hunks {
		hunks[i] = h.clone()
	}
	return &DMPPatch{
		LeftStart:   p.LeftStart,
		RightStart:  p.RightStart,
		LeftLength:  p.LeftLength,
		RightLength: p.RightLength,
		Hunks:       hunks,
	}
}

func (p *DMPPatch) headHunk() *DMPHunk          { return p.Hunks[0] }
func (p *DMPPatch) tailHunk() *DMPHunk          { return p.Hunks[len(p.Hunks)-1] }
func (p *DMPPatch) exceptHeadHunks() []*DMPHunk { return p.Hunks[1:] }

func (p *DMPPatch) leftLength() int {
	var n int
	for _, h := range p.Hunks {
		switch h.Op {
		case DMPOpDelete, DMPOpEqual:
			n += h.countLines()
		}
	}
	return n
}
func (p *DMPPatch) rightLength() int {
	var n int
	for _, h := range p.Hunks {
		switch h.Op {
		case DMPOpInsert, DMPOpEqual:
			n += h.countLines()
		}
	}
	return n
}

func (p *DMPPatch) header(color bool) string {
	x := fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
		p.LeftStart, p.LeftLength,
		p.RightStart, p.RightLength,
	)
	if color {
		return cyanString(x)
	}
	return x
}

func (p *DMPPatch) IntoString(color bool) string {
	xs := make([]string, len(p.Hunks))
	for i, h := range p.Hunks {
		xs[i] = h.IntoString(color)
	}
	return p.header(color) + strings.Join(xs, "")
}

func hunksToDebugString(hunks []*DMPHunk) string {
	ss := make([]string, len(hunks))
	for i, h := range hunks {
		ss[i] = h.debugString()
	}
	return strings.Join(ss, "")
}

func (p *DMPPatch) debugString() string { return hunksToDebugString(p.Hunks) }

type DMPResult struct {
	LeftLabel  string
	RightLabel string
	Patches    []*DMPPatch
}

func (r *DMPResult) header(color bool) string {
	left := "--- " + r.LeftLabel
	right := "+++ " + r.RightLabel
	if color {
		left = redString(left)
		right = greenString(right)
	}
	return left + "\n" + right + "\n"
}

func (r *DMPResult) IntoString(color bool) string {
	xs := make([]string, len(r.Patches))
	for i, x := range r.Patches {
		xs[i] = x.IntoString(color)
	}
	return r.header(color) + strings.Join(xs, "")
}

func patchesToDebugString(patches []*DMPPatch) string {
	ss := make([]string, len(patches))
	for i, p := range patches {
		ss[i] = p.debugString()
	}
	return strings.Join(ss, ",")
}

type DMP struct {
	LeftLabel  string
	RightLabel string
	Context    int
}

var ErrDMPNoDiff = errors.New("DMPNoDiff")

func (p *DMP) Diff(left, right string) (*DMPResult, error) {
	diffs := p.rawDiff(left, right)
	switch len(diffs) {
	case 0:
		return nil, ErrDMPNoDiff
	case 1:
		d := diffs[0]
		if d.Op == DMPOpEqual {
			return nil, ErrDMPNoDiff
		}
	}
	slog.Debug("diff: rawDiff", slog.Int("len", len(diffs)), slog.String("debug", hunksToDebugString(diffs)))

	patches, err := p.rawPatches(diffs)
	if err != nil {
		return nil, err
	}
	slog.Debug("diff: rawPatches", slog.Int("len", len(patches)), slog.String("debug", patchesToDebugString(patches)))

	if patches, err = p.mergePatches(patches); err != nil {
		return nil, err
	}
	slog.Debug("diff: mergePatches", slog.Int("len", len(patches)), slog.String("debug", patchesToDebugString(patches)))

	if patches, err = p.writePatchHeaders(patches); err != nil {
		return nil, err
	}
	slog.Debug("diff: writePatchHeaders", slog.String("debug", patchesToDebugString(patches)))

	return &DMPResult{
		LeftLabel:  p.LeftLabel,
		RightLabel: p.RightLabel,
		Patches:    patches,
	}, nil
}

var errWritePatchHeaders = errors.New("WritePatchHeaders")

func (p *DMP) writePatchHeaders(patches []*DMPPatch) ([]*DMPPatch, error) {
	var (
		result                = make([]*DMPPatch, len(patches))
		contextSize           = p.Context
		leftLinum, rightLinum int
	)
	for i, p := range patches {
		if len(p.Hunks) == 0 || len(p.Hunks) == 1 && p.Hunks[0].Op == DMPOpEqual {
			return nil, fmt.Errorf("invalid hunk: %#v: %w", p, errWritePatchHeaders)
		}

		cloned := p.clone()
		result[i] = cloned

		if p.headHunk().Op == DMPOpEqual {
			n := p.headHunk().countLines()
			cloned.headHunk().Body = p.headHunk().tail(contextSize)
			delta := n - contextSize + 1
			if delta < 0 {
				delta = 0
			}
			if i > 0 {
				// subtract extra rows of Equal from base linum
				// because given patches are like:
				// [Equal1, Insert1, Delete1, Equal2]
				// [Equal2, Insert2, Equal3]
				// [Equal3, ...]
				// ...
				// so overlapped Equals produce extra rows
				leftLinum -= n
				rightLinum -= n
			}
			cloned.LeftStart = leftLinum + delta
			cloned.RightStart = rightLinum + delta
		} else {
			cloned.LeftStart = leftLinum
			cloned.RightStart = rightLinum
		}
		if p.tailHunk().Op == DMPOpEqual {
			cloned.tailHunk().Body = p.tailHunk().head(contextSize)
		}

		if cloned.LeftStart == 0 && cloned.RightStart == 0 {
			// special adjustments are needed
			// because 0 means the patch contains no hunks of the file
			var (
				leftStart1, rightStart1 bool
			)
			for _, h := range cloned.Hunks {
				switch h.Op {
				case DMPOpDelete:
					leftStart1 = true
				case DMPOpInsert:
					rightStart1 = true
				default:
					leftStart1 = true
					rightStart1 = true
				}
			}
			if leftStart1 {
				cloned.LeftStart = 1
			}
			if rightStart1 {
				cloned.RightStart = 1
			}
		}

		cloned.LeftLength = cloned.leftLength()
		cloned.RightLength = cloned.rightLength()

		leftLinum += p.leftLength()
		rightLinum += p.rightLength()
	}

	return result, nil
}

var errMergePatches = errors.New("MergePatches")

// mergePatches merges overlapped hunks for diff context.
func (p *DMP) mergePatches(patches []*DMPPatch) ([]*DMPPatch, error) {
	var (
		result   []*DMPPatch
		curPatch *DMPPatch
		add      = func() {
			result = append(result, curPatch)
			curPatch = nil
		}
		prevPatch *DMPPatch

		contextSize = p.Context
	)
	for i, p := range patches {
		isTail := i == len(patches)-1
		if i > 0 {
			prevPatch = patches[i-1]
		}
		if prevPatch == nil {
			curPatch = p
			if isTail {
				add()
				break
			}
			continue
		}
		mkErr := func(s string) error {
			return fmt.Errorf("%s: prev=%#v cur=%#v: %w", s, prevPatch, p, errMergePatches)
		}
		if curPatch == nil {
			curPatch = p
			if isTail {
				add()
				break
			}
			continue
		}
		// want curPatch: [X, Y, ..., Equal]
		// want p:        [Equal, Z, ...]
		prevTailHunk := curPatch.tailHunk()
		curHeadHunk := p.headHunk()
		if prevTailHunk != curHeadHunk {
			return nil, mkErr("prevTailHunk != curHeadHunk")
		}
		if curHeadHunk.Op != DMPOpEqual {
			return nil, mkErr("prevTailHunk and curHeadHunk are not Equal")
		}
		if curHeadHunk.countLines() > 2*contextSize {
			// cannot merge curPatch and p
			// there are some rows between them that belongs to neither
			add()
			curPatch = p
			if isTail {
				add()
				break
			}
			continue
		}
		// merge curPatch and p
		curPatch.Hunks = append(curPatch.Hunks, p.exceptHeadHunks()...)
		if isTail {
			add()
			break
		}
	}

	return result, nil
}

var errRawPatches = errors.New("RawPatches")

// rawPatches builds the diff patches from the diff hunks.
// A diff patch has additional hunks for diff context.
func (p *DMP) rawPatches(hunks []*DMPHunk) ([]*DMPPatch, error) {
	var (
		patches  []*DMPPatch
		patch    *DMPPatch
		prevHunk *DMPHunk

		add = func() {
			patches = append(patches, patch)
			patch = nil
		}
		newPatch = func(hunk *DMPHunk) {
			patch = &DMPPatch{
				Hunks: []*DMPHunk{
					hunk,
				},
			}
		}
		push = func(hunk *DMPHunk) {
			if patch == nil {
				newPatch(hunk)
				return
			}
			patch.Hunks = append(patch.Hunks, hunk)
		}
	)
	for i, h := range hunks {
		isTail := i == len(hunks)-1
		if i > 0 {
			prevHunk = hunks[i-1]
		}
		curChanged := h.Op != DMPOpEqual
		if prevHunk == nil {
			if curChanged { // head is Insert or Delete
				push(h)
				if isTail {
					add()
					break
				}
			}
			continue
		}

		mkErr := func(s string) error {
			return fmt.Errorf("%s: prev=%#v cur=%#v: %w", s, prevHunk, h, errRawPatches)
		}

		prevChanged := prevHunk.Op != DMPOpEqual
		switch {
		case prevChanged && curChanged: // e.g. Delete -> Insert
			if patch == nil {
				return nil, mkErr("prevChanged curChanged but patch was nil")
			}
			push(h) // push next change
			if isTail {
				add()
				break
			}
			continue
		case prevChanged && !curChanged: // e.g. Insert -> Equal
			if patch == nil {
				return nil, mkErr("prevChanged not curChanged but patch was nil")
			}
			push(h) // push Equal for Context
			add()   // flush
			continue
		case !prevChanged && curChanged: // e.g. Equal -> Delete
			if patch != nil {
				return nil, mkErr("not prevChanged curChanged but patch was not nil")
			}
			push(prevHunk) // push Equal for Context
			push(h)
			if isTail {
				add()
				break
			}
			continue
		default: // Equal -> Equal
			return nil, mkErr("prevChanged and curChanged cannnot both be false")
		}
	}

	return patches, nil
}

// rawDiff calculates the diff hunks.
// They have all data of left and right,
// and no two Equal hunks are adjacent.
func (*DMP) rawDiff(left, right string) []*DMPHunk {
	dmp := diffmatchpatch.New()
	a, b, c := dmp.DiffLinesToChars(left, right)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, c)

	xs := make([]*DMPHunk, len(diffs))
	for i, d := range diffs {
		x := &DMPHunk{
			Body: d.Text,
		}
		switch d.Type {
		case diffmatchpatch.DiffDelete:
			x.Op = DMPOpDelete
		case diffmatchpatch.DiffInsert:
			x.Op = DMPOpInsert
		case diffmatchpatch.DiffEqual:
			x.Op = DMPOpEqual
		default:
			panic(fmt.Errorf("unreachable! DMP.rawDiff: diff=%#v", d))
		}
		xs[i] = x
	}
	return xs
}
