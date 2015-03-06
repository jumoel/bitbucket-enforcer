package main

import (
	"./gobucket"
	"./log"
	dotenv "./vendor/godotenv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	/*
	  "time"
	  "regexp"
	*/
	"strings"
)

type RepositorySettings struct {
	LandingPage  string
	Private      bool
	MainBranch   string
	Forks string
	DeployKeys   []struct {
		Name string
		Key  string
	}
	PostHooks        []string
	BranchManagement struct {
		PreventDelete []string
		PreventRebase []string
		AllowPushes   map[string]struct {
			Groups []string
			Users  []string
		}
	}

	AccessManagement struct {
		Users  []map[string]string // An array of username => permission maps
		Groups []map[string]string // ditto
	}
}

var configDir = flag.String("configdir", "configs", "the folder containing repository configrations")
var verbose = flag.Bool("v", false, "print more output")

func main() {
	log.SetPrefix("bitbucket-enforcer")

	flag.Parse()

	err := dotenv.Load()
	if err != nil {
		log.Notice(".env error", err)
	}

	bb_username := os.Getenv("BITBUCKET_ENFORCER_USERNAME")
	bb_key := os.Getenv("BITBUCKET_ENFORCER_API_KEY")

	gobucket := gobucket.New(bb_username, bb_key)

	/*
	  var enforcement_matcher = regexp.MustCompile(`-enforce(?:=([a-zA-Z0-9]+))?`)

	  var last_etag string = ""
	  var changed bool
		for _ = range time.Tick(1 * time.Second) {
			changed, last_etag = gobucket.RepositoriesChanged(bb_username, last_etag)

	    if !changed {
	      fmt.Println("No repository changes, sleeping.")
	      continue
	    }

	    repos := gobucket.GetRepositories(bb_username)

	    for _, repo := range repos {
	      if strings.Contains(repo.Description, "-noenforce") {
	        fmt.Printf("Skipping <%s> because of '-noenforce'\n", repo.FullName)
	        continue
	      }

	      if strings.Contains(repo.Description, "-enforced") {
	        fmt.Printf("Skipping <%s> because of '-enforced'\n", repo.FullName)
	        continue
	      }

	      matches := enforcement_matcher.FindStringSubmatch(repo.Description)

	      enforcement_policy := "default"
	      if len(matches) > 0 {
	        enforcement_policy = matches[1]
	      }

	      enforcePolicy(repo.FullName, enforcement_policy)
	    }
		}
	*/

	enforcePolicy("omi-nu/omi-test-nytnytnyt", "default")

	repo_fullname := "omi-nu/omi-test-nytnytnyt"
	policyname := "default"

	parts := strings.Split(repo_fullname, "/")
	policy := parseConfig(policyname)

	fmt.Println(gobucket.PutLandingPage(parts[0], parts[1], policy.LandingPage))
  fmt.Println(gobucket.PutPrivacy(parts[0], parts[1], policy.Private))
  fmt.Println(gobucket.PutForks(parts[0], parts[1], policy.Forks))
}

func enforcePolicy(repo_fullname string, policyname string) {

}

func parseConfig(configFile string) RepositorySettings {
	config_raw, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", *configDir, configFile))
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
