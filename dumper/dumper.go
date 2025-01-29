package dumper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/vault-thirteen/HttpStreamDumper/rwc"
	ae "github.com/vault-thirteen/auxie/errors"
)

const (
	Msg_DumpStart   = "Dumping has started. URL: "
	Msg_DumpStop    = "Dumping has stopped. URL: "
	MsgF_BytesSaved = "%v bytes were saved."
	Msg_Stopping    = "Stopping ..."
	Msg_Stopped     = "Stopped."
)

type Dumper struct {
	ctx      context.Context
	cancelFn context.CancelFunc

	url          string
	httpResponse *http.Response

	outputFilePath string
	outputFile     *os.File

	reader            io.Reader
	errors            chan error
	streamHasFinished chan bool
	wg                *sync.WaitGroup
}

func NewDumper(url string, outputFilePath string) (d *Dumper) {
	d = &Dumper{
		url:               url,
		outputFilePath:    outputFilePath,
		errors:            make(chan error, 8),
		streamHasFinished: make(chan bool, 1),
		wg:                new(sync.WaitGroup),
	}

	d.ctx, d.cancelFn = context.WithCancel(context.Background())

	return d
}

func (d *Dumper) Start() (err error) {
	d.outputFile, err = os.Create(d.outputFilePath)
	if err != nil {
		return err
	}

	d.httpResponse, err = http.Get(d.url)
	if err != nil {
		return err
	}

	d.reader = rwc.NewReaderWithContext(d.httpResponse.Body, d.ctx)

	d.wg.Add(1)
	go d.readStreamAsync()

	return nil
}

func (d *Dumper) Errors() (errChan chan error) {
	return d.errors
}

func (d *Dumper) StreamHasFinished() (x chan bool) {
	return d.streamHasFinished
}

func (d *Dumper) readStreamAsync() {
	defer d.wg.Done()

	log.Println(Msg_DumpStart, d.url)

	n, err := io.Copy(d.outputFile, d.reader)
	if err != nil {
		isSeriousError := !errors.Is(err, context.Canceled)
		if isSeriousError {
			d.errors <- err
		}
	} else {
		d.streamHasFinished <- true
	}

	log.Println(Msg_DumpStop, d.url)
	log.Println(fmt.Sprintf(MsgF_BytesSaved, n))
}

func (d *Dumper) Stop() (err error) {
	defer d.cancelFn()

	defer func() {
		derr := d.httpResponse.Body.Close()
		if derr != nil {
			err = ae.Combine(err, derr)
		}
	}()

	defer func() {
		derr := d.outputFile.Close()
		if derr != nil {
			err = ae.Combine(err, derr)
		}
	}()

	log.Println(Msg_Stopping)
	d.cancelFn()
	d.wg.Wait()
	close(d.errors)
	log.Println(Msg_Stopped)
	return nil
}
