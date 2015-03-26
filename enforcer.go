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

type branchManagement struct {
	PreventDelete []string
	PreventRebase []string
	AllowPushes   map[string]struct {
		Groups []string
		Users  []string
	}
}

type repositorySettings struct {
	LandingPage        string
	Private            interface{}
	MainBranch         string
	Forks              string
	PublicIssueTracker interface{}
	DeployKeys         publicKeyList
	PostHooks          []string
	BranchManagement   branchManagement

	AccessManagement struct {
		Users  []map[string]string // An array of username => permission maps
		Groups []map[string]string // ditto
	}
}

type publicKey struct {
	Name string
	Key  string
}

type publicKeyList []publicKey

type bbServices []gobucket.Service
type matchType int

const (
	matchNone matchType = iota
	matchContent
	matchExact
)

var configDir = flag.String("configdir", "configs", "the folder containing repository configrations")
var verbose = flag.Bool("v", false, "print more output")
var bbAPI *gobucket.APIClient

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
		fmt.Println(bbAPI.SetLandingPage(parts[0], parts[1], policy.LandingPage))
		fmt.Println(bbAPI.SetPrivacy(parts[0], parts[1], policy.Private))
		fmt.Println(bbAPI.SetForks(parts[0], parts[1], policy.Forks))
		fmt.Println(bbAPI.SetMainBranch(parts[0], parts[1], policy.MainBranch))
		fmt.Println(enforceDeployKeys(parts[0], parts[1], policy.DeployKeys))
		fmt.Println(bbAPI.GetServices(parts[0], parts[1]))
		fmt.Println(enforcePOSTHooks(parts[0], parts[1], policy.PostHooks))
		fmt.Println(enforceBranchManagement(parts[0], parts[1], policy.BranchManagement))
	*/
	if policy.PublicIssueTracker != nil {
		fmt.Println(bbAPI.SetPublicIssueTracker(parts[0], parts[1], policy.PublicIssueTracker.(bool)))
	}

	// Avoid errors about unused variables
	//fmt.Println(policy, parts)
}

func enforcePolicy(repoFullname string, policyname string) {

}

func enforceBranchManagement(owner string, repo string, policies branchManagement) error {
	for _, branch := range policies.PreventDelete {
		if err := bbAPI.AddBranchRestriction(owner, repo, "delete", branch, nil, nil); err != nil {
			return err
		}
	}

	for _, branch := range policies.PreventRebase {
		if err := bbAPI.AddBranchRestriction(owner, repo, "force", branch, nil, nil); err != nil {
			return err
		}
	}

	for branch, permissions := range policies.AllowPushes {
		if err := bbAPI.AddBranchRestriction(owner, repo, "push", branch, permissions.Users, permissions.Groups); err != nil {
			return err
		}
	}

	return nil
}

func (hooks *bbServices) hasPOSTHook(URL string) bool {
	for _, hook := range *hooks {
		if hook.Service.Type == "POST" {
			for _, field := range hook.Service.Fields {
				if field.Name == "URL" && field.Value == URL {
					return true
				}
			}
		}
	}

	return false
}

func enforcePOSTHooks(owner string, repo string, hookURLs []string) error {
	hookList, err := bbAPI.GetServices(owner, repo)

	if err != nil {
		return err
	}

	var currentHooks bbServices = hookList

	for _, url := range hookURLs {
		if !currentHooks.hasPOSTHook(url) {
			if err := bbAPI.AddService(owner, repo, "POST", map[string]string{"URL": url}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (keys *publicKeyList) hasKey(needle gobucket.DeployKey) (matchType, int) {
	for index, key := range *keys {
		if key.Key == needle.Key && key.Name == needle.Label {
			return matchExact, index
		} else if key.Key == needle.Key {
			return matchContent, index
		}
	}

	return matchNone, -1
}

/*
This method ensures the presence of all required keys.
- It removes keys with matching content but mismatching names. Afterwards they
  are added again, this time with the correct name.
- It adds keys that are not present.
- It doesn't remove keys that are present in Bitbucket but not in the policy
  file.
*/
func enforceDeployKeys(owner string, repo string, keys publicKeyList) error {
	currkeys, _ := bbAPI.GetDeployKeys(owner, repo)

	newkeys := make(publicKeyList, len(keys))
	copy(newkeys, keys)

	for _, key := range currkeys {
		match, matchIndex := newkeys.hasKey(key)

		if match == matchContent {
			// Delete the key from BB so it can be reuploaded with proper name
			if err := bbAPI.DeleteDeployKey(owner, repo, key.ID); err != nil {
				return err
			}
		} else if match == matchExact {
			// Don't waste time reuploading key as it is an exact match
			newkeys = append(newkeys[:matchIndex], newkeys[(matchIndex+1):]...)
		}
	}

	for _, key := range newkeys {
		if err := bbAPI.AddDeployKey(owner, repo, key.Name, key.Key); err != nil {
			return err
		}
	}

	return nil
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
