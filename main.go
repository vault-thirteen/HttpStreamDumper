package main

import (
	"os"
	"os/signal"
	"syscall"

	ver "github.com/vault-thirteen/auxie/Versioneer/classes/Versioneer"
	"github.com/vault-thirteen/auxie/cio"
	"github.com/vault-thirteen/auxie/errors"

	"github.com/vault-thirteen/HttpStreamDumper/dumper"
)

const (
	Msg_Usage              = "Usage: <app> <HTTP_Stream_URL> <Output_File>"
	Msg_ClosingApplication = "Closing application ..."
	Msg_StreamEnd          = "Stream has reached its end."
	Msg_DumpingError       = "Dumping error"

	Msgf_QuitSignalIsReceived = "Quit signal from O.S. has been received: %v"
)

const (
	Err_NotEnoughArguments = "not enough arguments"
)

func main() {
	vi, err := ver.New(false)
	if err != nil {
		errors.LogError(err)
		os.Exit(1)
	}
	showIntro(vi, "")

	var url, outFile string
	url, outFile, err = getCLA()
	if err != nil {
		showUsage()
		return
	}

	err = dumpHttpStream(url, outFile)
	if err != nil {
		errors.LogError(err)
		os.Exit(1)
	}
}

func showIntro(v *ver.Versioneer, serviceName string) {
	v.ShowIntroText(serviceName)
	v.ShowComponentsInfoText()
	cio.PrintMessage()
}

func getCLA() (url, outFile string, err error) {
	if len(os.Args) < 3 {
		return "", "", errors.ErrorFromText(Err_NotEnoughArguments)
	}

	url = os.Args[1]
	outFile = os.Args[2]

	return url, outFile, nil
}

func showUsage() {
	cio.PrintMessage(Msg_Usage)
}

func dumpHttpStream(url string, outputFilePath string) (err error) {
	d := dumper.NewDumper(url, outputFilePath)

	err = d.Start()
	if err != nil {
		return err
	}

	mustStop := make(chan bool, 1)
	osSignals := make(chan os.Signal, 8)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for sig := range osSignals {
			switch sig {
			case syscall.SIGINT,
				syscall.SIGTERM:
				cio.LogMessage(cio.FormatMessage(Msgf_QuitSignalIsReceived, sig))
				mustStop <- true
			}
		}
	}()

	// Wait for a signal or an error.
	select {
	case _ = <-mustStop:
		cio.PrintMessage(Msg_ClosingApplication)
	case dumpErr := <-d.Errors():
		errors.LogError(errors.ErrorWithPrefix(dumpErr, Msg_DumpingError))
	case _ = <-d.StreamHasFinished():
		cio.PrintMessage(Msg_StreamEnd)
		cio.PrintMessage(Msg_ClosingApplication)
	}

	err = d.Stop()
	if err != nil {
		return err
	}

	// Read other errors if there are any.
	for err = range d.Errors() {
		if err != nil {
			return err
		}
	}

	return nil
}
