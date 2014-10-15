package main

import (
  "os"
  "fmt"
  "io/ioutil"
  "log"
  "encoding/json"
  "flag"
)

type RepositorySettings struct {
  LandingPage string // TODO unmarshal into enum like type
  Private bool
  MainBranch string
  Forks string // TODO: Unmarshal 'forks' into an enum like type
  DeployKeys []struct {
    Name string
    Key string
  }
  PostHooks []string
  BranchManagement struct {
    PreventDelete []string
    PreventRebase []string
    AllowPushes []struct {
      BranchName string
      Groups []string
      Users []string
    }
  }

  AccessManagement struct {
    Users []struct {
      User string
      Permission string // TODO unmarshal permission into an enum like type (read, write, adming)
    }
    Groups []struct {
      Group string
      Permission string // TODO unmarshal permission into an enum like type (read, write, adming)
    }
  }
}

var configFile = flag.String("config", "golive.json", "the configfile to read")
var verbose = flag.Bool("v", false, "print more output")

func main() {
  oauth_key := os.Getenv("BITBUCKET_ENFORCER_KEY")
  oauth_pass := os.Getenv("BITBUCKET_ENFORCER_PASS")

  fmt.Println("key:", oauth_key)
  fmt.Println("pass:", oauth_pass)
}

func parseConfig(configFile string) RepositorySettings {
    config_raw, err := ioutil.ReadFile(configFile)
    if err != nil {
      log.Fatal(err)
    }

    var config RepositorySettings
    json.Unmarshal(config_raw, &config)

    if *verbose {
      log.Print("Loaded config: ", config)
    }

    return config
}
