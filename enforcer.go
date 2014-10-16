package main

import (
  _ "github.com/joho/godotenv/autoload"
  "os"
  "fmt"
  "io/ioutil"
  "log"
  "encoding/json"
  "flag"
  "./gobucket"
)

type RepositorySettings struct {
  LandingPage string
  Private bool
  MainBranch string
  Forks string
  DeployKeys []struct {
    Name string
    Key string
  }
  PostHooks []string
  BranchManagement struct {
    PreventDelete []string
    PreventRebase []string
    AllowPushes map[string] struct {
      Groups []string
      Users []string
    }
  }

  AccessManagement struct {
    Users []map[string] string // An array of username => permission maps
    Groups []map[string] string // ditto
  }
}

var configDir = flag.String("configdir", "configs", "the folder containing repository configrations")
var verbose = flag.Bool("v", false, "print more output")

func main() {
  flag.Parse()

  oauth_key := os.Getenv("BITBUCKET_ENFORCER_KEY")
  oauth_pass := os.Getenv("BITBUCKET_ENFORCER_PASS")

  fmt.Println("key:", oauth_key)
  fmt.Println("pass:", oauth_pass)

  settings := parseConfig(*configDir + "/default.json")

  res2B, _ := json.Marshal(settings)
  fmt.Println(string(res2B))
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
