package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	// get home dir
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// validate args
	if len(os.Args[1:]) == 0 {
		fmt.Println("Error")
		os.Exit(0)
	}

	// get arguments
	args := dropFlags(os.Args[1:])

	// init
	if args[0] == "init" {

		if err := os.Mkdir(homedir+"/.watch-client/", os.ModePerm); err != nil {
			log.Fatal(err)
		}

		if err := downloadFile(homedir+"/.watch-client/.env", "https://raw.githubusercontent.com/Mr-MSA/Watch/main/.env"); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		if err := downloadFile(homedir+"/.watch-client/structure.json", "https://raw.githubusercontent.com/Mr-MSA/Watch/main/structure.json"); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	}

	// validate baseurl
	if envVariable("baseURL") == "WATCH_SERVER" {
		fmt.Println("Please set watch server address at ~/.watch-client/.env")
		os.Exit(0)
	}

	// read and parse configurations
	config := ReadJSON(homedir + "/.watch-client/structure.json")

	// show help
	showHelp(args, config)

	// set variables
	var api string
	var count = 1

	// find endpoint
	var conf map[string]interface{} = config
	for i := 0; i <= len(args); i++ {

		if conf[args[i]] == nil {
			break
		}

		if fmt.Sprintf("%T", conf[args[i]]) == "string" {
			api = conf[args[i]].(string)
			count += i
			break
		} else {
			conf = conf[args[i]].(map[string]interface{})
		}
	}

	// validate api
	if api == "" {
		fmt.Println("API not found")
		os.Exit(0)
	}

	// check has arg
	if strings.Contains(api, "{{arg}}") {
		count++
	}

	// parse api endpoint
	api = strings.Replace(api, "{{arg}}", args[len(args)-1], -1)
	api = strings.Replace(api, "{{base}}", envVariable("baseURL"), -1)

	// parse flags
	flags := os.Args[1:][count:]
	var flagArgs intelArgs

	intelCommand := flag.NewFlagSet("main", flag.ContinueOnError)

	defineIntelArgumentFlags(intelCommand, &flagArgs)

	if err := intelCommand.Parse(flags); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(2)
	}

	// set body
	var body string
	if flagArgs.Body != "" {

		body = flagArgs.Body

	} else if flagArgs.BodyFile != "" {

		// read body file
		fileContent, err := ioutil.ReadFile(flagArgs.BodyFile)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(0)
		}

		body = string(fileContent)
	}

	// append ?
	if !strings.Contains(api, "?") {
		api = api + "?"
	}

	// set endpoint by flags
	if flagArgs.JSON {
		api = fmt.Sprintf("%s&json=true", api)
	}

	if flagArgs.Provider != "" {
		api = fmt.Sprintf("%s&provider=%s", api, flagArgs.Provider)
	}

	// set default request methods
	if flagArgs.Method == "" {
		switch args[0] {
		case "regexp":
			if args[1] == "test" {
				flagArgs.Method = "POST"
			} else if args[1] == "apply" {
				flagArgs.Method = "PUT"
			}
		case "orch":
			flagArgs.Method = "PATCH"
		case "put":
			flagArgs.Method = "PATCH"
		case "delete":
			flagArgs.Method = "DELETE"
		default:
			flagArgs.Method = "GET"
		}
	}

	// set loop
	if args[0] == "get" {
		if args[1] == "lives" || args[1] == "fresh" || args[1] == "subdomains" || args[1] == "latest" {
			flagArgs.Loop = true
		}
	}

	if flagArgs.Count {
		api = fmt.Sprintf("%s&count=true", api)
	}

	if flagArgs.CDN {
		api = fmt.Sprintf("%s&cdn=true", api)
	}

	if flagArgs.Total {
		api = fmt.Sprintf("%s&total=true", api)
	}

	// limit res
	if flagArgs.Limit {
		flagArgs.Loop = false
	}

	var out string
	if flagArgs.Loop && !flagArgs.Count {

		// get count of results
		count, err := strconv.Atoi(MakeHttpRequest(api+"&count=true", flagArgs, body))
		if err != nil {
			fmt.Println("Can't convert string to integer")
		}

		// get results
		for i := 0; i <= ((count / 1000) + 1); i++ {
			resp := MakeHttpRequest(api+"&limit=1000&page="+strconv.Itoa(i), flagArgs, body)

			if flagArgs.Compare == "" {
				fmt.Print(resp)
			} else {
				out += resp

			}
		}

	} else {

		// send http request to api endpoint
		resp := MakeHttpRequest(api, flagArgs, body)
		if flagArgs.Compare == "" {
			fmt.Print(resp)
		} else {
			out = resp
		}
	}

	if flagArgs.Compare != "" {

		f, err := os.Create(".tmp")
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()
		_, err2 := f.WriteString(out)
		if err2 != nil {
			log.Fatal(err2)
		}

		var f1, f2 string
		if flagArgs.ReverseCompare {
			f1 = flagArgs.Compare
			f2 = ".tmp"

		} else {
			f1 = ".tmp"
			f2 = flagArgs.Compare
		}

		if _, err := os.Stat(flagArgs.Compare); err != nil {
			fmt.Println("Compare file does not exist!")
			return
		}

		cmd := exec.Command("grep", "-Fxvf", f1, f2)
		stdout, err := cmd.Output()

		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Print(string(stdout))
	}

}