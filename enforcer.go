package main

import (
  "os"
  "fmt"
)

func main() {
  oauth_key := os.Getenv("BITBUCKET_ENFORCER_KEY")
  oauth_pass := os.Getenv("BITBUCKET_ENFORCER_PASS")

  fmt.Println("key:", oauth_key)
  fmt.Println("pass:", oauth_pass)
}
