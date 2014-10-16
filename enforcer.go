package main

import (
  dotenv "./vendor/godotenv"
  "os"
  "fmt"
  "io/ioutil"
  "./log"
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
  log.SetPrefix("bitbucket-enforcer")

  flag.Parse()

  err := dotenv.Load()
  if (err != nil) {
    log.Notice(".env error", err)
  }

  oauth_key := os.Getenv("BITBUCKET_ENFORCER_KEY")
  oauth_pass := os.Getenv("BITBUCKET_ENFORCER_PASS")

  fmt.Println("key:", oauth_key)
  fmt.Println("pass:", oauth_pass)

  gobucket := gobucket.New("", "")
  fmt.Println(gobucket)

//  settings := parseConfig(*configDir + "/default.json")
}

func parseConfig(configFile string) RepositorySettings {
    config_raw, err := ioutil.ReadFile(configFile)
    if err != nil {
      log.Panic(err)
    }

    var config RepositorySettings
    json.Unmarshal(config_raw, &config)

    if *verbose {
      log.Info("Loaded config: ", config)
    }

    return config
}
