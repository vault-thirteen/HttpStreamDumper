package rwc

import (
	"context"
	"io"
)

type ReaderWithContext struct {
	rdr io.Reader
	ctx context.Context
}

func NewReaderWithContext(r io.Reader, c context.Context) io.Reader {
	return &ReaderWithContext{rdr: r, ctx: c}
}

func (r *ReaderWithContext) Read(dst []byte) (n int, err error) {
	err = r.ctx.Err()
	if err != nil {
		return 0, err
	}

	return r.rdr.Read(dst)
}
