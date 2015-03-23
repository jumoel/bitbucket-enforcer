package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jumoel/bitbucket-enforcer/gobucket"
	"github.com/jumoel/bitbucket-enforcer/log"
	dotenv "github.com/jumoel/bitbucket-enforcer/vendor/godotenv"
)

type keyList map[string]string

type repositorySettings struct {
	LandingPage      string
	Private          bool
	MainBranch       string
	Forks            string
	DeployKeys       keyList
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
var bbAPI *gobucket.ApiClient

func main() {
	log.SetPrefix("bitbucket-enforcer")

	flag.Parse()

	err := dotenv.Load()
	if err != nil {
		log.Notice(".env error", err)
	}

	bbUsername := os.Getenv("BITBUCKET_ENFORCER_USERNAME")
	bbKey := os.Getenv("BITBUCKET_ENFORCER_API_KEY")

	bbAPI = gobucket.New(bbUsername, bbKey)

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

	repoFullname := "omi-nu/omi-test-nytnytnyt"
	policyname := "default"

	parts := strings.Split(repoFullname, "/")
	policy := parseConfig(policyname)
	/*
		fmt.Println(bbapi.PutLandingPage(parts[0], parts[1], policy.LandingPage))
		fmt.Println(bbapi.PutPrivacy(parts[0], parts[1], policy.Private))
		fmt.Println(bbapi.PutForks(parts[0], parts[1], policy.Forks))
		fmt.Println(bbapi.PutMainBranch(parts[0], parts[1], policy.MainBranch))

	*/

	enforceDeployKey(parts[0], parts[1], policy.DeployKeys)
}

func enforcePolicy(repoFullname string, policyname string) {

}

func enforceDeployKey(owner string, repo string, keys keyList) {
	currkeys, _ := bbAPI.GetDeployKeys(owner, repo)

	/*
		for index, key := range currkeys {
			// Check if key already exists
		}
	*/
	fmt.Printf("%+v\n", currkeys)
	fmt.Printf("%+v\n", keys)
}

func parseConfig(configFile string) repositorySettings {
	rawConfig, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", *configDir, configFile))
	if err != nil {
		log.Panic(err)
	}

	var config repositorySettings
	json.Unmarshal(rawConfig, &config)

	if *verbose {
		log.Info("Loaded config: ", config)
	}

	return config
}
