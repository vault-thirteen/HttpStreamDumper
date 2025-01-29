package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vault-thirteen/HttpStreamDumper/dumper"
	"github.com/vault-thirteen/auxie/Versioneer"
)

const (
	MsgF_QuitSignalIsReceived = "Quit signal from O.S. has been received: %v"
	Msg_Usage                 = "Usage: <app> <HTTP_Stream_URL> <Output_File>"
	Msg_ClosingApplication    = "Closing application ..."
	Msg_StreamEnd             = "Stream has reached its end."
	Msg_DumpingError          = "Dumping error:"
)

const (
	Err_NotEnoughArguments = "not enough arguments"
)

func main() {
	vi, err := ver.New()
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
}

func showIntro(v *ver.Versioneer, serviceName string) {
	v.ShowIntroText(serviceName)
	v.ShowComponentsInfoText()
	fmt.Println()
}

func getCLA() (url, outFile string, err error) {
	if len(os.Args) < 3 {
		return "", "", errors.New(Err_NotEnoughArguments)
	}

	url = os.Args[1]
	outFile = os.Args[2]

	return url, outFile, nil
}

func showUsage() {
	fmt.Println(Msg_Usage)
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
				log.Println(fmt.Sprintf(MsgF_QuitSignalIsReceived, sig))
				mustStop <- true
			}
		}
	}()

	// Wait for a signal or an error.
	select {
	case _ = <-mustStop:
		fmt.Println(Msg_ClosingApplication)
	case dumpErr := <-d.Errors():
		fmt.Println(Msg_DumpingError, dumpErr)
	case _ = <-d.StreamHasFinished():
		fmt.Println(Msg_StreamEnd)
		fmt.Println(Msg_ClosingApplication)
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
