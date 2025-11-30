package main

import (
    "os"
    // "fmt"
    "log"
    "flag"
    "github.com/zombielifestyle/crackdown/crackdown"
)

var file string
func init() {
    flag.StringVar(&file, "file", "", "file to parse")
    flag.Parse()
}

func main() {
    f, err := os.Open(file)
    if err != nil {
        log.Fatalf("cannot open %q:\n%s\n", file, err)
    }

    s := crackdown.ConvertFile(f)
    os.Stdout.Write(s)
}

