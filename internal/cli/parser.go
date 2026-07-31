package cli

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
)

func (f *Flags) Parser() error {
	args_length := len(os.Args)
	args := os.Args[1:]

	if args_length < 3 {

		slog.Error("[CLI::Parser] : We need atleast 2 Argument :")
		Usage()
		os.Exit(1)
	}
	for i := 0; i < args_length-1; i++ {
		switch args[i] {
		case "-url", "-u", "-U":
			if i+1 > args_length-1 {
				slog.Error("[CLI::Parser]: Error there is no Link")
				return fmt.Errorf("No link")
			}

			link := args[i+1]
			if !func() bool {
				url_parser, err := url.Parse(link) // better approach then the Regex usage
				if err != nil {
					return false
				}
				if url_parser.Host == " " {
					return false
				}
				return true
			}() {

				slog.Error("[CLI::Parser]: Incorrect Link Format !!")
				return fmt.Errorf("Incorrect Link Format")
			}
			f.Url_link = link
			i++

		case "-concurrent", "-c", "-C":
			if i+1 > args_length-1 {
				slog.Error("[CLI::Parser]: Error, there is no Concurrent Value provided")
				Usage()
				return fmt.Errorf("No Concurrent Value")
			}
			conc_n, err := strconv.Atoi(args[i+1])
			if err != nil {
				slog.Error("[CLI::Parser]: Concurrent  is not the integer", slog.Any("error", err))
				return fmt.Errorf("Conccurrent not integer")
			}
			if conc_n < 0 && conc_n > 9 {
				slog.Error("[CLI::Parser]: Not a valid Concurrent Input!!")
				Usage()
				return fmt.Errorf("Not a Valid Input")
			}

			f.Concurrent_n = conc_n
			i++
		case "-output", "-o", "-O":
			if i+1 > args_length-1 {
				slog.Error("[CLI::Parser]: There is no output Path")
				Usage()
				return fmt.Errorf("No output Path")
			}
			filePath := args[i+1]
			if filePath == "" || filePath == "/" {
				slog.Error("[CLI::Parser]: Invalid or Prohibited FilePath")
				Usage()
				return fmt.Errorf("Prohibited FilePath")
			}
			f.Filepath = filePath
			i++

		case "help":
			if i+1 <= args_length-1 {
				if i+1+1 <= len(args)-1 {
					//REQUIRED TO BE PRINT-ED
					fmt.Printf("No help for '%v' \n", args[i+1]+args[i+1+1])
					return fmt.Errorf("Usage Helper function  called!!")
				}

				if args[i+1] == "concurrent" {
					con_usage()
				} else if args[i+1] == "url" {
					url_usage()
				} else {
					fmt.Println("There is no such Sub-command")
					slog.Error("[CLI::Parser]: Illegal  subcommand in help !!")

				}
				i++
				return fmt.Errorf("Subcommand Usasge helper function called!!")
			}

			Usage()
			return fmt.Errorf("Usage helper function called !!")

		default:
			slog.Info(fmt.Sprintf("[CLI::Parser]: Unknown Flags!! %v", args[i]))
			fmt.Printf("%v : unkown command \nRun 'tandem help' for usage.", args[i])
			return fmt.Errorf("Unknown Command")
			//Usage()
		}

	}

	return nil
}
