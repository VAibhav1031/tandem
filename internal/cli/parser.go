package cli

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
)

func (f *Flags) Parser() error {
	args_length := len(os.Args)
	args := os.Args[1:]

	if args_length < 3 {

		slog.Error("Error : We need atleast 2 Argument :")
		Usage()
		os.Exit(1)
	}
	for i := 0; i < args_length; i++ {

		switch args[i] {

		case "-url", "-URL", "-u", "-U":
			if i+1 >= args_length {
				slog.Error("Error there is no Link")
				return fmt.Errorf("No link")
			}
			link := args[i+1]
			if !func() bool {
				linkRegex, err := regexp.Compile(`^http?://[^\s$.?#].[^\s]*$`)
				if err != nil {
					return false
					//log
					// exit
				}
				return linkRegex.MatchString(link)
			}() {

				slog.Error("Incorrect Link Format !!")
				return fmt.Errorf("Incorrect Link Format")
			}
			f.Url_link = link

		case "-concurrent", "-CONCURRENT", "-c", "-C":
			if i+1 >= args_length {
				slog.Error("Error, there is no Concurrent Value provided")
				Usage()
				return fmt.Errorf("No Concurrent Value")
			}
			conc_n, err := strconv.Atoi(args[i+1])
			if err != nil {
				slog.Error("Concurrent  is not the integer", err)
				return fmt.Errorf("Conccurrent not integer")
			}
			if conc_n < 0 && conc_n > 9 {
				slog.Error("Not a valid Concurrent Input!!")
				Usage()
				return fmt.Errorf("Not a Valid Input")
			}

			f.Concurrent_n = conc_n
		case "-OUTPUT", "-output", "-o", "-O":
			if i+1 >= args_length {
				slog.Error("There is no output Path")
				Usage()
				return fmt.Errorf("No output Path")
			}
			filePath := args[i+1]
			if filePath == "" || filePath == "/" || filePath == "" {
				slog.Error("Error: Invalid or Prohibited FilePath")
				Usage()
				return fmt.Errorf("Prohibited FilePath")
			}
			f.Filepath = filePath
		default:
			slog.Error("Unknown Flags!!, %v", args[i])
			Usage()
		}

	}

	return nil
}
