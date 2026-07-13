package cli

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
)

func (f *Flags) Parser() {
	args_length := len(os.Args)
	args := os.Args[1:]
	if args_length > 1 && args[0] == "--setup" {
		slog.Info("Initial Setup Started")
		RunSetup()
		return
	}

	if args_length < 3 {

		fmt.Println("Error : We need atleast 2 Argument :")
		Usage()
		os.Exit(1)
	}
	for i := 0; i < args_length; i++ {

		switch args[i] {

		case "-url", "-URL", "-u", "-U":
			if i+1 >= args_length {
				fmt.Println("Error there is no Link")
				return
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

				fmt.Println("Error ")
			}
			f.Url_link = link

		case "-concurrent", "-CONCURRENT", "-c", "-C":
			if i+1 >= args_length {
				fmt.Println("Error, there is no Concurrent Value provided")
				Usage()
				return
			}
			conc_n, err := strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Println("Error : It is not the integer", err)
				return
			}
			if conc_n < 0 && conc_n > 9 {
				fmt.Println("Error: Not a valid Concurrent Input!!")
				Usage()
				return
			}

			f.Concurrent_n = conc_n
		case "-OUTPUT", "-output", "-o", "-O":
			if i+1 >= args_length {
				fmt.Println("Error: There is no Link")
				Usage()
				return
			}
			filePath := args[i+1]
			if filePath == "" || filePath == "/" || filePath == "" {
				fmt.Println("Error: Invalid or Prohibited FilePath")
				Usage()
				return
			}
			f.Filepath = filePath
		default:
			fmt.Printf("Unknown Flags!!, %v", args[i])
			Usage()
		}

	}
}
