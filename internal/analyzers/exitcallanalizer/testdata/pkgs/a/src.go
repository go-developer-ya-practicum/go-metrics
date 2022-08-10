package main

import "os"

func main() {
	os.Exit(0) // want `call os.Exit\(\) in main function of main package`
}
