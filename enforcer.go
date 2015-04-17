package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

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

type accessManagement struct {
	Users  map[string]string // usernames => permissions
	Groups map[string]string // groupnames => permissions
}

type repositorySettings struct {
	LandingPage      string
	Private          interface{}
	Forks            string
	IssueTracker     string
	DeployKeys       publicKeyList
	PostHooks        []string
	BranchManagement branchManagement
	AccessManagement accessManagement
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

const sleepTime = 1 * time.Second

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

	scanRepositories(bbUsername)
}

func scanRepositories(bbUsername string) {
	var enforcementMatcher = regexp.MustCompile(`-enforce(?:=([a-zA-Z0-9]+))?`)

	var lastEtag string
	var changed bool

	for _ = range time.Tick(sleepTime) {
		var err error
		if changed, lastEtag, err = bbAPI.RepositoriesChanged(bbUsername, lastEtag); err != nil {
			log.Error(fmt.Sprintf("Error determining if repository list has changed (%s)", err))
		}

		if !changed {
			if *verbose {
				log.Info("No repository changes, sleeping.")
			}
			continue
		}

		log.Info("Repository list changed")

		repos, err := bbAPI.GetRepositories(bbUsername)

		if err != nil {
			log.Error("Error getting repository list", err)
			continue
		}

		for _, repo := range repos {
			if strings.Contains(repo.Description, "-noenforce") {
				if *verbose {
					log.Info(fmt.Sprintf("Skipping <%s> because of '-noenforce'\n", repo.FullName))
				}
				continue
			}

			if strings.Contains(repo.Description, "-enforced") {
				if *verbose {
					log.Info(fmt.Sprintf("Skipping <%s> because of '-enforced'\n", repo.FullName))
				}
				continue
			}

			matches := enforcementMatcher.FindStringSubmatch(repo.Description)

			enforcementPolicy := "default"
			if len(matches) > 0 {
				enforcementPolicy = matches[1]
			}

			log.Info(fmt.Sprintf("Enforcing repo '%s' with policy '%s'", repo.FullName, enforcementPolicy))

			parts := strings.Split(repo.FullName, "/")
			err := enforcePolicy(parts[0], parts[1], enforcementPolicy)

			if err != nil {
				log.Warning(fmt.Sprintf("Could not enforce policy '%s' on repo '%s'. Will be processed again next cycle. (%s)", enforcementPolicy, repo.FullName, err))
			} else {
				newDescription := strings.TrimSpace(fmt.Sprintf("%s\n\n-enforced", repo.Description))

				if err := bbAPI.SetDescription(parts[0], parts[1], newDescription); err != nil {
					log.Warning("Could not set description on repo '%s'. Will be processed again next cycle.", repo.FullName)
				}
			}
		}
	}
}

func enforcePolicy(owner string, repo string, policyname string) error {
	policy, err := parseConfig(policyname)

	if err != nil {
		log.Error(fmt.Sprintf("Error parsing parsing policy '%s': ", policyname), err)
		return err
	}

	if policy.Private != nil {
		if err := bbAPI.SetPrivacy(owner, repo, policy.Private.(bool)); err != nil {
			log.Warning("Error setting privacy: ", err)
			return err
		}
	}

	if policy.Forks != "" {
		if err := bbAPI.SetForks(owner, repo, policy.Forks); err != nil {
			log.Warning("Error fork policy: ", err)
			return err
		}
	}

	if policy.LandingPage != "" {
		if err := bbAPI.SetLandingPage(owner, repo, policy.LandingPage); err != nil {
			log.Warning("Error setting landing page: ", err)
			return err
		}
	}

	if len(policy.DeployKeys) > 0 {
		if err := enforceDeployKeys(owner, repo, policy.DeployKeys); err != nil {
			log.Warning("Error setting deploy keys: ", err)
			return err
		}
	}

	if len(policy.PostHooks) > 0 {
		if err := enforcePOSTHooks(owner, repo, policy.PostHooks); err != nil {
			log.Warning("Error setting POST hooks: ", err)
			return err
		}
	}

	if policy.IssueTracker != "" {
		if err := bbAPI.SetIssueTracker(owner, repo, policy.IssueTracker); err != nil {
			log.Warning("Error setting issue tracker: ", err)
			return err
		}
	}

	if err := enforceBranchManagement(owner, repo, policy.BranchManagement); err != nil {
		log.Warning("Error setting branch policies: ", err)
		return err
	}

	if err := enforceAccessManagement(owner, repo, policy.AccessManagement); err != nil {
		log.Warning("Error setting access policies: ", err)
		return err
	}

	return nil
}

func enforceAccessManagement(owner string, repo string, policies accessManagement) error {
	for username, privilege := range policies.Users {
		if err := bbAPI.AddUserPrivilege(owner, repo, username, privilege); err != nil {
			return err
		}
	}

	for groupname, privilege := range policies.Groups {
		if err := bbAPI.AddGroupPrivilege(owner, repo, groupname, privilege); err != nil {
			return err
		}
	}

	return nil
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

func parseConfig(configFile string) (repositorySettings, error) {
	rawConfig, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", *configDir, configFile))
	if err != nil {
		return repositorySettings{}, err
	}

	var config repositorySettings
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return repositorySettings{}, err
	}

	if *verbose {
		log.Info("Loaded config: ", config)
	}

	return config, nil
}
