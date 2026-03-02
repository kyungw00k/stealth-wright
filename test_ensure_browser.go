//go:build ignore

package main

import (
    "fmt"
    "time"
    
    "github.com/kyungw00k/sw/internal/browserutil"
)

func main() {
    fmt.Println("Testing EnsureBrowser...")
    start := time.Now()
    
    browser, err := browserutil.EnsureBrowser("chromium")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Browser: %s (took %v)\n", browser, time.Since(start))
}
